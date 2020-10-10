# fastFS
An alternative golang static file server used memory cache and file handle reuse.

## example:
```
fs := fastFS.NewServer("/webroot")
```
### or
```
fs := &fastFS.Server{
		Fs: &fastFS.FileSystem{
			Root:       "/webroot",
			MemQuota:   128 * fastFS.MB,
			CacheLimit: 1 * fastFS.MB,
		},
	}
```
### install into http server
```
http.Handle("/", fs)
log.Fatal(http.ListenAndServe(":80", nil))

```
