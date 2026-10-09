// Package api는 openapi.yaml에서 생성한 서버 인터페이스와 모델을 담는다.
package api

//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml
