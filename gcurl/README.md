# gcurl
A Go HTTP client library for creating and sending API requests

## Examples

```go
import "github.com/sonnt85/gosutils/gcurl"
```

### Basic request

```go
req := gcurl.NewRequest(nil)

// GET
resp, err := req.Get("http://example.com/api/users")

if err != nil {
	log.Fatalln("Unable to make request: ", err)
}

fmt.Println(resp.Text())

// POST
user := &User{...}

resp, err := req.Post("http://example.com/api/users", user)

if err != nil {
	log.Fatalln("Unable to make request: ", err)
}

fmt.Println(resp.Text())
```

### Chained request

```go
user := newUser()

req := gcurl.NewRequest()

resp, err := req.WithBasicAuth("admin", "passwd").WithHeader("x-trace-id", "123").Post("http://example.com/api/users")

if err != nil {

	log.Fatalln("Unable to make request: ", err)

}

fmt.Println(resp.Text())

```
