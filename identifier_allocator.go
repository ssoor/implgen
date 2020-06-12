package main

import "strconv"

type identifierAllocator map[string]struct{}

func newIdentifierAllocator(taken []string) identifierAllocator {
	a := make(identifierAllocator, len(taken))
	for _, s := range taken {
		a[s] = struct{}{}
	}
	return a
}

func (o identifierAllocator) allocateIdentifier(want string) string {
	id := want
	for i := 2; ; i++ {
		if _, ok := o[id]; !ok {
			o[id] = struct{}{}
			return id
		}
		id = want + "_" + strconv.Itoa(i)
	}
}
