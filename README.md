# godi

Small, type safe dependency injection library for go >= 1.19


[![Apache 2.0 license](https://img.shields.io/badge/license-apache2-brightgreen.svg)](https://opensource.org/licenses/Apache-2.0)
[![test status](https://github.com/streamonkey/godi/actions/workflows/tests.yml/badge.svg?branch=main "test status")](https://github.com/streamonkey/godi/actions)

## service definition

Services (a service can be anything) are registered on the container build as simple factory functions and referenced via a unique service id. Service ids are typed strings referencing the referenced service type.


```go

package main

import (
    "github.com/streamonkey/godi"
)

type (
    Config struct {
        ValueA string
    }
)

const (
    MyValue godi.ServiceID[string] = "my.value"
)

func MyValueFactory(ctx context.Context, c *godi.Container[*Config]) (string, error) {
    return c.Config().ValueA, nil
}


func main() {
    containerBuilder := godi.New(&Config{ValueA: "abc"})
    godi.Register(containerBuilder, ServiceA, createServiceA)
}

```

## service initialization

Before services can be retreived, a container must be build from the container builder. Once the container is built, services can no longer be registered on the container builder.

Services can be retreived from the container using `godi.Get`. 
Dependency resolution within factories works as well.


```go
package main

import (
    "log"
    "github.com/streamonkey/godi"
)

type (
    Config struct { AName string, Bname string }
    ServiceA struct { Name string }
    ServiceB struct { Name string, A *ServiceA }
)


const (
    ServiceAID godi.ServiceID[*ServiceA] = "service.A"
    ServiceBID godi.ServiceID[*ServiceB] = "service.B"
)


func createServiceA(ctx context.Context, c *godi.Container[*Config]) (*ServiceA, error) {
    return &ServiceA{Name: c.Config().AName}, nil
}

func createServiceB(ctx context.Context, c *godi.Container[*Config]) (*ServiceB, error) {
    serviceA, err := godi.Get(ctx, c, ServiceAID)
    if err != nil {
        return nil, err
    }
    return &ServiceB{Name: c.Config().Bname, A: serviceA}, nil
}

func main() {
    conf := &Config{}

    containerBuilder := godi.New(conf)
    godi.Register(containerBuilder, ServiceAID, createServiceA)
    godi.Register(containerBuilder, ServiceBID, createServiceB)
    
    container, err := containerBuilder.Build()
    if err != nil {
        log.Fatal(err)
    }
}
```

## wraping service resolution errors

For traceability reasons it may be useful to wrap service resolution errors in a deticated error type



```go 
package main

import (
    "github.com/streamonkey/godi/resolve"
)

type (
    Service struct {}
)

const (
    MyServiceID godi.ServiceID[*Service] = "my.service"
)

func newService() (*Service, error) {...}

func createMyService(ctx context.Context, c *godi.Container[*Config]) (*Service, error) {
    s, err  := newService()
    if err != nil {
        return nil, resolve.Error(MyServiceID, err)
    }
    
    return s, nil
}
```