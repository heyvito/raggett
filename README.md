# Raggett

> **Raggett** is an opinionated Go HTTP Server Framework

## Installing

```
go get github.com/heyvito/raggett@v0.1.0
```

## Usage

Raggett is built on top of [Chi](https://github.com/go-chi/chi) to provide
routing and middlewares; the way it handles requests and responses are the
main difference between the common way Go HTTP servers are built. Instead of
implementing handlers dealing with both `http.Request` and
`http.ResponseWriter`, the library encapsulates all common operations on a
single `Request` object. The library also makes heavy usage of [Zap](https://github.com/uber-go/zap)
to provide logging facilities.

The following snippet represents a simple Raggett Application:

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/heyvito/raggett"
    "go.uber.org/zap"
)

type HelloRequest struct {
    *raggett.Request
    Name string `form:"name" required:"true" blank:"false"`
}

func main() {
    mux := raggett.NewMux(zap.L())

    // Handlers must be a function accepting a single struct that uses
    // *raggett.Request as a promoted field, and returns an error.
    mux.Post("/", func(r HelloRequest) error {
        r.RespondString(fmt.Sprintf("Hello, %s!", r.Name))
        return nil
    })

    http.ListenAndServe(":3000", mux)
}
```


The same application can also be extended to provide both HTML and plain text
responses by implementing a structure implementing a set of interfaces provided
by the library. For instance:

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/heyvito/raggett"
    "go.uber.org/zap"
)

type HelloRequest struct {
    *raggett.Request
    Name string `form:"name" required:"true" blank:"false"`
}

type HelloResponse struct {
    Name string
}

func (h HelloResponse) JSON() interface{} {
    return map[string]interface{}{
        "greeting": fmt.Sprintf("Hello, %s!", h.Name),
    }
}

func (h HelloResponse) HTML() string {
    return fmt.Sprintf("<h1>Hello, %s!</h1>", h.Name)
}

func main() {
    mux := raggett.NewMux(zap.L())
    mux.Post("/", func(r HelloRequest) error {
        r.Respond(HelloResponse{
            Name: r.Name,
        })
        return nil
    })

    http.ListenAndServe(":3000", mux)
}
```

## Accessing Form Values

When defining a request object, form values can be automatically loaded and
converted to specific types by using tags:

```go
type SignUpRequest struct {
    *raggett.Request
    Name                string `form:"name" required:"true" blank:"false"`
    Email               string `form:"email" pattern:"^.+@.+\..+$"`
    SubscribeNewsletter bool   `form:"subscribe"`
}
```

## Accessing QueryString Values

The same pattern used by forms can be applied to QueryString parameters:

```go
type ListUsersRequest struct {
    *raggett.Request
    Page int `query:"page"`
}
```

## Accessing URL Parameters

As Raggett is built on top of Chi, URL parameters can also be accessed through
fields defined on structs and marked with tags:

```go
type ListPostsForDay struct {
    *raggett.Request
    Day   int `url-param:"day"`
    Month int `url-param:"month"`
    Year  int `url-param:"year"`
}

mux.Get("/blog/posts/{day}/{month}/{year}", func(r ListPostsForDay) error {
        // ...
    })
```

## Parsing Request Bodies
When not using Forms or Multipart requests, applications can also rely on
JSON or XML being posted, for instance. For that, Raggett has a set of Resolvers
that can be attached directly to a field indicating that the request's body
must be parsed and set it:

```go

type SignUpRequest struct {
    *raggett.Request
    UserData struct {
        Email string `json:"email"`
        Name  string `json:"name"`
    } `body:"json"`
}
```

## Receiving Files

Multipart data is also supported. To receive a single file:

```go
type ExampleRequest struct {
    *raggett.Request
    Photo *raggett.FileHeader `form:"photo" required:"false"`
}
```

Or multiple files:

```go
type ExampleRequest struct {
    *raggett.Request
    Photos []*raggett.FileHeader `form:"photo" required:"false"`
}
```

> **ProTip™:** `raggett.FileHeader` is simply an alias to stdlib's
`multipart.FileHeader`. Both types are interchangeable on Raggett.


## Accessing Headers
Just like other values, headers can also be obtained through tags:

```go
type AuthenticatedRequest struct {
    *raggett.Request
    Authorization string `header:"authorization" required:"true"`
}
```

## Defaults

Raggett provides default handlers for errors such as validation (HTTP 400),
runtime (HTTP 500), Not Found (HTTP 404), and Method Not Allowed (HTTP 405). For
those errors, the library is capable of responding to the following formats,
based on the `Accept` header provided by the client:

- HTML
- JSON
- XML
- Plain Text

The library also provides a "Development" mode, which augments information
provided by those error handlers:

```go
mux := raggett.NewMux(...)
mux.Development = true
```

> :warning: Warning! Setting Development to `true` on production environments is
unadvised, since it may cause sensitive information to be exposed to the
internet.

## License

```
The MIT License (MIT)

Copyright (c) 2021 Victor Gama de Oliveira

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```
