package tests

import (
    "github.com/goal-web/contracts"
    "github.com/goal-web/filesystem"
    "github.com/stretchr/testify/assert"
    "os"
    "testing"
)

// Integration test for OSS adapter; requires endpoint, bucket, and credentials via env.
func TestOSSAdapter(t *testing.T) {
    endpoint := os.Getenv("OSS_ENDPOINT")
    accessKeyID := os.Getenv("OSS_ACCESS_KEY_ID")
    accessKeySecret := os.Getenv("OSS_ACCESS_KEY_SECRET")
    bucket := os.Getenv("OSS_BUCKET")

    if endpoint == "" || accessKeyID == "" || accessKeySecret == "" || bucket == "" {
        t.Skip("skip OSS integration test: require OSS_ENDPOINT, OSS_ACCESS_KEY_ID, OSS_ACCESS_KEY_SECRET, OSS_BUCKET")
        return
    }

    var factory = filesystem.New(filesystem.Config{
        Default: "oss",
        Disks: map[string]contracts.Fields{
            "oss": {
                "driver":            "oss",
                "endpoint":          endpoint,
                "bucket":            bucket,
                "access_key_id":     accessKeyID,
                "access_key_secret": accessKeySecret,
                // optional: "domain": "https://your-cdn-domain",
                // optional: "private": true,
            },
        },
    })

    disk := factory.Disk("oss")
    // basic CRUD
    key := "goalweb/test/hello.txt"
    assert.Nil(t, disk.Put(key, "goal"))
    assert.True(t, disk.Exists(key))
    content, err := disk.Get(key)
    assert.Nil(t, err)
    assert.Equal(t, "goal", content)
    assert.Nil(t, disk.Delete(key))
}