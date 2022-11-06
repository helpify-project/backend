package cctx

import "context"

func WithValues(parent context.Context, values ...interface{}) (ctx context.Context) {
	if len(values)%2 != 0 {
		panic("uneven")
	}

	ctx = parent
	for i := 0; i < len(values); i++ {
		key := values[i]
		value := values[i+1]
		i++

		ctx = context.WithValue(ctx, key, value)
	}
	return
}
