## Install

    go get github.com/sonnt85/gosutils/smerge

    // use in your .go code
    import (
        "github.com/sonnt85/gosutils/smerge"
    )

## Usage

You can only merge same-type structs with exported fields initialized as zero value of their type and same-types maps. Mergo won't merge unexported (private) fields but will do recursively any exported one. It won't merge empty structs value as [they are zero values](https://golang.org/ref/spec#The_zero_value) too. Also, maps will be merged recursively except for structs inside maps (because they are not addressable using Go reflection).

```go
if err := smerge.Merge(&dst, src); err != nil {
    // ...
}
```

Also, you can merge overwriting values using the transformer `WithOverride`.

```go
if err := smerge.Merge(&dst, src, smerge.WithOverride); err != nil {
    // ...
}
```

Additionally, you can map a `map[string]interface{}` to a struct (and otherwise, from struct to map), following the same restrictions as in `Merge()`. Keys are capitalized to find each corresponding exported field.

```go
if err := smerge.Map(&dst, srcMap); err != nil {
    // ...
}
```

Warning: if you map a struct to map, it won't do it recursively. Don't expect Mergo to map struct members of your struct as `map[string]interface{}`. They will be just assigned as values.

Here is a nice example:

```go
package main

import (
	"fmt"
	"github.com/sonnt85/gosutils/smerge"
)

type Foo struct {
	A string
	B int64
}

func main() {
	src := Foo{
		A: "one",
		B: 2,
	}
	dest := Foo{
		A: "two",
	}
	smerge.Merge(&dest, src)
	fmt.Println(dest)
	// Will print
	// {two 2}
}
```

Note: if test are failing due missing package, please execute:

    go get gopkg.in/yaml.v2

### Transformers

Transformers allow to merge specific types differently than in the default behavior. In other words, now you can customize how some types are merged. For example, `time.Time` is a struct; it doesn't have zero value but IsZero can return true because it has fields with zero value. How can we merge a non-zero `time.Time`?

```go
package main

import (
	"fmt"
	"github.com/sonnt85/gosutils/smerge"
        "reflect"
        "time"
)

type timeTransformer struct {
}

func (t timeTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ == reflect.TypeOf(time.Time{}) {
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				isZero := dst.MethodByName("IsZero")
				result := isZero.Call([]reflect.Value{})
				if result[0].Bool() {
					dst.Set(src)
				}
			}
			return nil
		}
	}
	return nil
}

type Snapshot struct {
	Time time.Time
	// ...
}

func main() {
	src := Snapshot{time.Now()}
	dest := Snapshot{}
	smerge.Merge(&dest, src, smerge.WithTransformers(timeTransformer{}))
	fmt.Println(dest)
	// Will print
	// { 2018-01-12 01:15:00 +0000 UTC m=+0.000000001 }
}
```

## Contact me

If I can help you, you have an idea or you are using Mergo in your projects, don't hesitate to drop me a line (or a pull request): [@im_dario](https://twitter.com/im_dario)

## About

Written by [Dario Castañé](http://dario.im).

## License

[BSD 3-Clause](http://opensource.org/licenses/BSD-3-Clause) license, as [Go language](http://golang.org/LICENSE).

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fimdario%2Fsmerge.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fimdario%2Fsmerge.ref=badge_large)
