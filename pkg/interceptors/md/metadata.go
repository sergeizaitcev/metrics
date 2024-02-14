package md

import (
	"context"

	"google.golang.org/grpc/metadata"
)

const (
	keyRealIP  = "real_ip"
	keyHash256 = "hash_256"
)

// SetRealIP устанавливает в контекст IP-адрес.
func SetRealIP(ctx context.Context, ip string) context.Context {
	return setKey(ctx, keyRealIP, ip)
}

// GetRealIP возвращает IP-адрес из контекста.
func GetRealIP(ctx context.Context) string {
	return getKey(ctx, keyRealIP)
}

// SetHash256 устанавливает в контекст hash256.
func SetHash256(ctx context.Context, hash string) context.Context {
	return setKey(ctx, keyHash256, hash)
}

// GetHash256 возвращает hash из контекста.
func GetHash256(ctx context.Context) string {
	return getKey(ctx, keyHash256)
}

func setKey(ctx context.Context, key, value string) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	md.Delete(key)
	md.Set(key, value)
	return metadata.NewOutgoingContext(ctx, md)
}

func getKey(ctx context.Context, key string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
