package adapters

import (
    "bufio"
    "github.com/aliyun/aliyun-oss-go-sdk/oss"
    "github.com/goal-web/contracts"
    "github.com/goal-web/filesystem/file"
    "github.com/goal-web/supports/logs"
    "github.com/goal-web/supports/utils"
    "io"
    "io/fs"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "strings"
    "time"
)

// OssAdapter constructs an Aliyun OSS filesystem adapter
func OssAdapter(name string, config contracts.Fields) contracts.FileSystem {
    var (
        endpoint  = utils.GetStringField(config, "endpoint")
        bucket    = utils.GetStringField(config, "bucket")
        ak        = utils.GetStringField(config, "access_key_id")
        sk        = utils.GetStringField(config, "access_key_secret")
        domain    = utils.GetStringField(config, "domain")
        private   = utils.GetBoolField(config, "private")
        ttl       = time.Duration(utils.GetIntField(config, "ttl")) * time.Second
    )
    client, err := oss.New(endpoint, ak, sk)
    if err != nil {
        logs.WithError(err).Error("OssAdapter: create client failed")
        panic(err)
    }
    bkt, err := client.Bucket(bucket)
    if err != nil {
        logs.WithError(err).Error("OssAdapter: get bucket failed")
        panic(err)
    }
    return &Oss{
        name:       name,
        endpoint:   endpoint,
        bucketName: bucket,
        client:     client,
        bucket:     bkt,
        domain:     domain,
        private:    private,
        ttl:        ttl,
    }
}

type Oss struct {
    name       string
    endpoint   string
    bucketName string
    client     *oss.Client
    bucket     *oss.Bucket
    domain     string
    private    bool
    ttl        time.Duration
}

func (o *Oss) Name() string { return o.name }

func (o *Oss) Exists(path string) bool {
    exist, err := o.bucket.IsObjectExist(path)
    if err != nil {
        logs.WithError(err).WithField("path", path).Debug("Oss.Exists: error")
        return false
    }
    return exist
}

func (o *Oss) Get(path string) (string, error) {
    bytes, err := o.Read(path)
    return string(bytes), err
}

func (o *Oss) Read(path string) ([]byte, error) {
    rc, err := o.bucket.GetObject(path)
    if err != nil {
        return nil, err
    }
    defer rc.Close()
    return io.ReadAll(rc)
}

func (o *Oss) ReadStream(path string) (*bufio.Reader, error) {
    rc, err := o.bucket.GetObject(path)
    if err != nil {
        return nil, err
    }
    return bufio.NewReader(rc), nil
}

func (o *Oss) Put(path, contents string) error {
    return o.bucket.PutObject(path, strings.NewReader(contents))
}

func (o *Oss) WriteStream(path string, contents string) error {
    return o.bucket.PutObject(path, strings.NewReader(contents))
}

// Url 生成可下载链接：
// - 若为私有桶，返回带签名的限时下载链接（默认 ttl 或 3600s）
// - 若为公开桶，优先使用配置的 domain，否则使用 bucket.endpoint 组合
func (o *Oss) Url(path string) string {
    key := trimLeadingSlash(path)
    if o.private {
        expires := int64(o.ttl.Seconds())
        if expires <= 0 {
            expires = 3600 // 默认 1 小时
        }
        url, err := o.bucket.SignURL(key, oss.HTTPGet, expires)
        if err != nil {
            logs.WithError(err).WithField("key", key).Debug("Oss.Url: sign failed")
            return ""
        }
        return url
    }
    if o.domain != "" {
        return strings.TrimRight(o.domain, "/") + "/" + key
    }
    return "https://" + o.bucketName + "." + o.endpoint + "/" + key
}

// RefreshUrlIfExpired 检测链接是否过期；若过期或接近过期（在 refreshThreshold 秒内），生成并返回新的链接，否则返回原链接
// 支持 V1（Expires）与 V4（x-oss-date + x-oss-expires）两种签名格式判断
// refreshThreshold: 在过期前多少秒内进行刷新（默认为 0，即仅在过期时刷新）
func (o *Oss) RefreshUrlIfExpired(link string, refreshThreshold int64) string {
    u, err := url.Parse(link)
    if err != nil {
        // 非法URL，直接返回原样，以避免误判
        return link
    }
    q := u.Query()
    now := time.Now()

    // V1：查询参数包含 Expires（Unix 秒）
    if expStr := q.Get("Expires"); expStr != "" {
        if exp, err := strconv.ParseInt(expStr, 10, 64); err == nil {
            if now.Unix() >= exp || now.Unix() >= exp-refreshThreshold {
                // 过期，基于路径重新生成；保留原有非签名参数
                key := strings.TrimLeft(u.Path, "/")
                if o.private {
                    opts := o.collectSignOptions(q)
                    expires := int64(o.ttl.Seconds())
                    if expires <= 0 {
                        expires = 3600
                    }
                    newURL, err := o.bucket.SignURL(key, oss.HTTPGet, expires, opts...)
                    if err != nil {
                        logs.WithError(err).WithField("key", key).Debug("Oss.RefreshUrlIfExpired: sign failed")
                        return ""
                    }
                    return newURL
                }
                // 公开：保留所有原查询参数
                base := o.publicBaseURL(key)
                if q.Encode() != "" {
                    return base + "?" + q.Encode()
                }
                return base
            }
            return link
        }
    }

    // V4：查询参数包含 x-oss-date（形如 20060102T150405Z）与 x-oss-expires（秒）
    if v4ExpStr := q.Get("x-oss-expires"); v4ExpStr != "" {
        if dateStr := q.Get("x-oss-date"); dateStr != "" {
            if base, err := time.Parse("20060102T150405Z", dateStr); err == nil {
                if v4Exp, err := strconv.ParseInt(v4ExpStr, 10, 64); err == nil {
                    if now.After(base.Add(time.Duration(v4Exp) * time.Second)) || now.After(base.Add(time.Duration(v4Exp-refreshThreshold) * time.Second)) {
                        key := strings.TrimLeft(u.Path, "/")
                        if o.private {
                            // 过期，按原参数重新签名
                            opts := o.collectSignOptions(q)
                            expires := int64(o.ttl.Seconds())
                            if expires <= 0 {
                                expires = 3600
                            }
                            newURL, err := o.bucket.SignURL(key, oss.HTTPGet, expires, opts...)
                            if err != nil {
                                logs.WithError(err).WithField("key", key).Debug("Oss.RefreshUrlIfExpired: v4 sign failed")
                                return ""
                            }
                            return newURL
                        }
                        // 公开：保留所有原查询参数
                        baseURL := o.publicBaseURL(key)
                        if q.Encode() != "" {
                            return baseURL + "?" + q.Encode()
                        }
                        return baseURL
                    }
                    return link
                }
            }
        }
    }

    // 无过期参数（可能为公开直链或其他场景），直接返回
    return link
}

// publicBaseURL 构造公开访问的基础URL（不含查询）
func (o *Oss) publicBaseURL(key string) string {
    if o.domain != "" {
        return strings.TrimRight(o.domain, "/") + "/" + key
    }
    return "https://" + o.bucketName + "." + o.endpoint + "/" + key
}

// collectSignOptions 从查询参数提取可签名的选项，确保新签名链接保留原有参数行为
func (o *Oss) collectSignOptions(q url.Values) []oss.Option {
    opts := make([]oss.Option, 0)
    if v := q.Get("x-oss-process"); v != "" {
        opts = append(opts, oss.Process(v))
    }
    // x-oss-traffic-limit 仅部分SDK版本支持，此处不直接签入；
    // 如需限速，建议通过CDN或在生成链接处另行处理。
    if v := q.Get("response-content-language"); v != "" {
        opts = append(opts, oss.ResponseContentLanguage(v))
    }
    if v := q.Get("response-expires"); v != "" {
        opts = append(opts, oss.ResponseExpires(v))
    }
    if v := q.Get("response-cache-control"); v != "" {
        opts = append(opts, oss.ResponseCacheControl(v))
    }
    if v := q.Get("response-content-disposition"); v != "" {
        opts = append(opts, oss.ResponseContentDisposition(v))
    }
    if v := q.Get("response-content-encoding"); v != "" {
        opts = append(opts, oss.ResponseContentEncoding(v))
    }
    if v := q.Get("response-content-type"); v != "" {
        opts = append(opts, oss.ResponseContentType(v))
    }
    return opts
}

func (o *Oss) GetVisibility(path string) contracts.FileVisibility {
    if o.private {
        return file.INVISIBLE
    }
    return file.VISIBLE
}

// SetVisibility: OSS doesn't support per-object ACL change via this adapter; noop
func (o *Oss) SetVisibility(path string, perm fs.FileMode) error { return nil }

func (o *Oss) Prepend(path, contents string) error {
    raw, _ := o.Get(path)
    return o.Put(path, contents+raw)
}

func (o *Oss) Append(path, contents string) error {
    raw, _ := o.Get(path)
    return o.Put(path, raw+contents)
}

func (o *Oss) Delete(path string) error {
    return o.bucket.DeleteObject(path)
}

func (o *Oss) Copy(from, to string) error {
    _, err := o.bucket.CopyObject(from, to)
    return err
}

func (o *Oss) Move(from, to string) error {
    if err := o.Copy(from, to); err != nil {
        return err
    }
    return o.Delete(from)
}

func (o *Oss) Size(path string) (int64, error) {
    hdr, err := o.bucket.GetObjectMeta(path)
    if err != nil {
        return 0, err
    }
    sizeStr := hdr.Get("Content-Length")
    if sizeStr == "" {
        return 0, nil
    }
    size, err := strconv.ParseInt(sizeStr, 10, 64)
    if err != nil {
        return 0, err
    }
    return size, nil
}

func (o *Oss) LastModified(path string) (time.Time, error) {
    hdr, err := o.bucket.GetObjectMeta(path)
    if err != nil {
        return time.Time{}, err
    }
    lm := hdr.Get("Last-Modified")
    if lm == "" {
        return time.Time{}, nil
    }
    t, err := http.ParseTime(lm)
    if err != nil {
        return time.Time{}, err
    }
    return t, nil
}

func (o *Oss) Files(directory string) []contracts.File {
    var files []contracts.File
    res, err := o.bucket.ListObjects(oss.Prefix(trimLeadingSlash(directory)))
    if err != nil {
        logs.WithError(err).WithField("dir", directory).Debug("Oss.Files: list failed")
        return files
    }
    for _, obj := range res.Objects {
        // skip directories themselves (OSS uses keys ending with "/" to represent folders)
        if strings.HasSuffix(obj.Key, "/") {
            continue
        }
        files = append(files, &OSSFile{
            DiskName: o.name,
            key:      obj.Key,
            disk:     o,
            OssFileInfo: OssFileInfo{
                name:    obj.Key,
                size:    obj.Size,
                modTime: obj.LastModified,
                isDir:   false,
            },
        })
    }
    return files
}

func (o *Oss) AllFiles(directory string) []contracts.File {
    var (
        files   []contracts.File
        marker  string
        prefix  = oss.Prefix(trimLeadingSlash(directory))
    )
    for {
        res, err := o.bucket.ListObjects(prefix, oss.Marker(marker))
        if err != nil {
            logs.WithError(err).WithField("dir", directory).Debug("Oss.AllFiles: list failed")
            break
        }
        for _, obj := range res.Objects {
            if strings.HasSuffix(obj.Key, "/") {
                continue
            }
            files = append(files, &OSSFile{
                DiskName: o.name,
                key:      obj.Key,
                disk:     o,
                OssFileInfo: OssFileInfo{
                    name:    obj.Key,
                    size:    obj.Size,
                    modTime: obj.LastModified,
                    isDir:   false,
                },
            })
        }
        if res.IsTruncated {
            marker = res.NextMarker
        } else {
            break
        }
    }
    return files
}

func (o *Oss) Directories(directory string) []string {
    var dirs []string
    res, err := o.bucket.ListObjects(
        oss.Prefix(trimLeadingSlash(directory)),
        oss.Delimiter("/"),
    )
    if err != nil {
        logs.WithError(err).WithField("dir", directory).Debug("Oss.Directories: list failed")
        return dirs
    }
    for _, p := range res.CommonPrefixes {
        dirs = append(dirs, p)
    }
    return dirs
}

func (o *Oss) AllDirectories(directory string) []string {
    // For OSS, listing with delimiter returns first-level directories; recursively collect
    var result []string
    queue := []string{trimLeadingSlash(directory)}
    for len(queue) > 0 {
        cur := queue[0]
        queue = queue[1:]
        res, err := o.bucket.ListObjects(
            oss.Prefix(cur),
            oss.Delimiter("/"),
        )
        if err != nil {
            logs.WithError(err).WithField("dir", cur).Debug("Oss.AllDirectories: list failed")
            continue
        }
        for _, p := range res.CommonPrefixes {
            result = append(result, p)
            queue = append(queue, p)
        }
    }
    return result
}

func (o *Oss) MakeDirectory(path string) error {
    // OSS directories are virtual; creating a placeholder object for compatibility
    if !strings.HasSuffix(path, "/") {
        path = path + "/"
    }
    return o.bucket.PutObject(trimLeadingSlash(path), strings.NewReader(""))
}

func (o *Oss) DeleteDirectory(directory string) error {
    // Delete all objects under the prefix
    var keys []string
    res, err := o.bucket.ListObjects(oss.Prefix(trimLeadingSlash(directory)))
    if err != nil {
        return err
    }
    for _, obj := range res.Objects {
        keys = append(keys, obj.Key)
    }
    if len(keys) == 0 {
        return nil
    }
    _, err = o.bucket.DeleteObjects(keys)
    return err
}

type OSSFile struct {
    DiskName string
    key      string
    disk     *Oss
    OssFileInfo
}

func (f *OSSFile) Read() []byte {
    bytes, _ := f.disk.Read(f.key)
    return bytes
}

func (f *OSSFile) ReadString() string {
    contents, _ := f.disk.Get(f.key)
    return contents
}

func (f *OSSFile) Disk() string { return f.DiskName }

func trimLeadingSlash(s string) string {
    return strings.TrimLeft(s, "/")
}

// OssFileInfo implements fs.FileInfo for OSS objects
type OssFileInfo struct {
    name    string
    size    int64
    modTime time.Time
    isDir   bool
}

func (o OssFileInfo) Name() string       { return o.name }
func (o OssFileInfo) Size() int64        { return o.size }
func (o OssFileInfo) Mode() fs.FileMode  { return os.ModePerm }
func (o OssFileInfo) ModTime() time.Time { return o.modTime }
func (o OssFileInfo) IsDir() bool        { return o.isDir }
func (o OssFileInfo) Sys() any           { return nil }