# fastmsgpack

[![Go Reference](https://pkg.go.dev/badge/github.com/hexon/fastmsgpack.svg)](https://pkg.go.dev/github.com/hexon/fastmsgpack)

Fastmsgpack is a Golang msgpack decoder. It is fast, but lacks features you might need. It can be used in combination with other msgpack libraries for e.g. encoding.

### Pros:

* It is very fast to query a list of fields from a nested msgpack structure.
* It is zero copy for strings and []byte.
* No reflection usage when decoding.

### Cons:

* It can only encode Go builtin types.
* The return value might contain pointers to the original data, so you can't modify the input data until you're done with the return value.
* It uses unsafe (to cast []byte to string without copying).
* It can't deserialize into your structs.
* It only supports strings as map keys.
* It decodes all ints as a Go `int`, including 64 bit ones, so it doesn't work on 32-bit platforms.

## Supported extensions:

* It supports extension -1 and decodes such values into a time.Time.
* It supports extension -128 (interned strings). I didn't find any documentation for it, but I've based it on https://github.com/vmihailenco/msgpack. You can pass a dict to the decoder and it will replace indexes into that dict with the strings from the dict.
* It introduces extension 17 which wraps map and arrays. Because extension headers contain the byte-length, partial decoding can efficiently skip over those values.
* It introduces extension 18 which is like a Switch statement inside the data. (Whether that's a good idea is up for debate.) We use this to pack data for multiple locales in one value.
* It introduces extension 19 which encodes void. When decoding a void as for example a map value the key and value are treated as non-existent.

## Returned types

`Decode` returns an `any`, which is either `int`, `float32`, `float64`, `string`, `[]byte`, `[]any`, `map[string]any` or `time.Time`.

`(*Resolver).Resolve` returns a list of such `any`s, one for each field requested.

## Example

```
r := fastmsgpack.NewResolver([]string{"person.properties.firstName", "person.properties.age"}, nil)
found, err := r.Resolve(data)
firstName, ok := found[0].(string)
age, ok := found[1].(int)
```
