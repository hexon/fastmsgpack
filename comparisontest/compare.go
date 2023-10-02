package main

import (
	"bytes"
	"log"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/hexon/fastmsgpack"
	"github.com/vmihailenco/msgpack/v5"
)

func main() {
	dec := msgpack.NewDecoder(os.Stdin)
	dec.ResetDict(os.Stdin, nil)

	subdec := msgpack.NewDecoder(nil)

	for {
		buf, err := dec.DecodeRaw()
		if err != nil {
			log.Fatalf("Failed to consume next: %v", err)
		}

		var want any
		subdec.ResetDict(bytes.NewReader(buf), nil)
		subdec.UseLooseInterfaceDecoding(true)
		upErr := subdec.Decode(&want)

		got, ourErr := fastmsgpack.Decode(buf, nil)

		if upErr != nil && ourErr != nil {
			log.Fatalf("Both decoders refused the input:\n%v\n%v", upErr, ourErr)
		}
		if upErr != nil {
			log.Fatalf("github.com/vmihailenco/msgpack refused this input, but we didn't: %v", upErr)
		}
		if ourErr != nil {
			log.Fatalf("we refused this input, but github.com/vmihailenco/msgpack didn't: %v", ourErr)
		}

		want = squashTypes(want)

		if diff := cmp.Diff(want, got); diff != "" {
			log.Fatalf("Disagreement.\n%s", diff)
		}
	}
}

func squashTypes(n any) any {
	switch n := n.(type) {
	case map[string]any:
		for k, v := range n {
			n[k] = squashTypes(v)
		}
	case []any:
		for i, v := range n {
			n[i] = squashTypes(v)
		}
	case uint64:
		return int(n)
	case int64:
		return int(n)
	}
	return n
}
