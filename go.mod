module github.com/go-rod/rod

go 1.15

require (
	github.com/gorilla/websocket v1.4.2
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.2-0.20200818115829-54d05a4e1844
	github.com/tidwall/gjson v1.6.0
	github.com/tidwall/sjson v1.1.1
	github.com/ysmood/goob v0.2.3
	github.com/ysmood/leakless v0.5.7
	go.uber.org/goleak v1.1.10
)

replace go.uber.org/goleak => github.com/ysmood/goleak v1.2.0
