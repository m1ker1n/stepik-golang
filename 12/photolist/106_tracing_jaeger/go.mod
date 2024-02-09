module photolist

go 1.13

require (
	github.com/99designs/gqlgen v0.10.1
	github.com/BurntSushi/toml v1.3.2 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.1.2 // indirect
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a
	github.com/aws/aws-sdk-go v1.25.31
	github.com/codahale/hdrhistogram v0.0.0-00010101000000-000000000000 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/disintegration/imaging v1.6.1
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/protobuf v1.3.2
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/golang-lru v0.5.0
	github.com/opentracing/opentracing-go v1.1.0
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749
	github.com/shurcooL/vfsgen v0.0.0-20181202132449-6a9ea43bcacd
	github.com/spf13/viper v1.5.0
	github.com/uber/jaeger-client-go v2.22.1+incompatible
	github.com/uber/jaeger-lib v2.2.0+incompatible
	github.com/vektah/gqlparser v1.1.2
	golang.org/x/crypto v0.0.0-20191029031824-8986dd9e96cf
	golang.org/x/image v0.0.0-20190802002840-cff245a6509b
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sys v0.0.0-20190801041406-cbf593c0f2f3
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/appengine v1.4.0
	google.golang.org/grpc v1.23.0
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.25.1

replace sourcegraph.com/sourcegraph/appdash-data => github.com/sourcegraph/appdash-data v0.0.0-20151005221446-73f23eafcf67

replace github.com/codahale/hdrhistogram => ./local/hdrhistogram-go-1.1.2
