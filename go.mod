module github.com/liujitcn/kratos-core

go 1.27.0

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.12-20260709200747-435963d16310.1
	buf.build/go/protovalidate v1.3.0
	github.com/go-kratos/kratos/v3 v3.0.0
	github.com/go-sql-driver/mysql v1.10.0
	github.com/google/wire v0.7.0
	github.com/liujitcn/go-utils v0.0.38
	github.com/liujitcn/go-utils/geoip v0.0.5
	github.com/liujitcn/go-utils/translator v0.0.4
	github.com/liujitcn/kratos-core/api v0.0.3
	github.com/liujitcn/kratos-kit v0.0.75
	github.com/liujitcn/kratos-kit/api v0.0.32
	github.com/liujitcn/kratos-kit/auth v0.0.25
	github.com/liujitcn/kratos-kit/auth/authn v0.0.23
	github.com/liujitcn/kratos-kit/auth/authn/engine/jwt v0.0.18
	github.com/liujitcn/kratos-kit/auth/authn/middleware v0.0.19
	github.com/liujitcn/kratos-kit/auth/authz v0.0.22
	github.com/liujitcn/kratos-kit/auth/authz/engine/casbin v0.0.20
	github.com/liujitcn/kratos-kit/auth/authz/middleware v0.0.18
	github.com/liujitcn/kratos-kit/bootstrap v0.0.23
	github.com/liujitcn/kratos-kit/cache v0.0.19
	github.com/liujitcn/kratos-kit/database/gorm v0.0.40
	github.com/liujitcn/kratos-kit/database/gorm/migration v0.0.13
	github.com/liujitcn/kratos-kit/oss v0.0.17
	github.com/liujitcn/kratos-kit/pprof v0.0.14
	github.com/liujitcn/kratos-kit/queue v0.0.27
	github.com/liujitcn/kratos-kit/server/grpc v0.0.2
	github.com/liujitcn/kratos-kit/server/http v0.0.3
	github.com/liujitcn/kratos-kit/server/mcp v0.0.2
	github.com/liujitcn/kratos-kit/server/sse v0.0.2
	github.com/liujitcn/kratos-kit/swagger-ui v0.0.15
	github.com/liujitcn/kratos-kit/translator v0.0.5
	github.com/liujitcn/kratos-kit/transport/cron v0.0.17
	github.com/liujitcn/kratos-kit/transport/mcp v0.0.14
	github.com/liujitcn/kratos-kit/transport/queue v0.0.3
	github.com/liujitcn/kratos-kit/transport/sse v0.0.13
	github.com/liujitcn/kratos-kit/utils v0.0.20
	github.com/mileusna/useragent v1.3.5
	github.com/nicksnyder/go-i18n/v2 v2.0.2
	github.com/robfig/cron/v3 v3.0.1
	golang.org/x/text v0.41.0
	google.golang.org/grpc v1.83.1
	google.golang.org/protobuf v1.36.12
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/gorm v1.31.2
	gorm.io/plugin/soft_delete v1.2.1
)

require (
	cel.dev/expr v0.25.2 // indirect
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/auth v0.18.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/longrunning v0.8.0 // indirect
	cloud.google.com/go/translate v1.12.7 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/ClickHouse/ch-go v0.61.5 // indirect
	github.com/ClickHouse/clickhouse-go/v2 v2.30.0 // indirect
	github.com/alibabacloud-go/alibabacloud-gateway-spi v0.0.5 // indirect
	github.com/alibabacloud-go/alimt-20190107 v1.0.0 // indirect
	github.com/alibabacloud-go/darabonba-openapi/v2 v2.1.15 // indirect
	github.com/alibabacloud-go/debug v1.0.1 // indirect
	github.com/alibabacloud-go/endpoint-util v1.1.0 // indirect
	github.com/alibabacloud-go/openapi-util v0.1.0 // indirect
	github.com/alibabacloud-go/tea v1.4.0 // indirect
	github.com/alibabacloud-go/tea-utils v1.4.4 // indirect
	github.com/alibabacloud-go/tea-utils/v2 v2.0.7 // indirect
	github.com/aliyun/aliyun-oss-go-sdk v3.0.2+incompatible // indirect
	github.com/aliyun/credentials-go v1.4.5 // indirect
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/aws/aws-sdk-go-v2 v1.43.2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.15 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.32.33 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.32 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.33 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.22.37 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.33 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.33 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.34 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.26 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.33 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.34 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.106.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.5.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.33.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.38.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.45.2 // indirect
	github.com/aws/smithy-go v1.27.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/bsm/redislock v0.9.4 // indirect
	github.com/casbin/casbin/v2 v2.135.0 // indirect
	github.com/casbin/govaluate v1.3.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clbanning/mxj/v2 v2.7.0 // indirect
	github.com/clipperhouse/displaywidth v0.10.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/form/v4 v4.3.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/cel-go v0.30.0 // indirect
	github.com/google/gnostic v0.7.1 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.12 // indirect
	github.com/googleapis/gax-go/v2 v2.17.0 // indirect
	github.com/gorilla/handlers v1.5.2 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/grafana/pyroscope-go v1.2.8 // indirect
	github.com/grafana/pyroscope-go/godeltaprof v0.1.9 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.6 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jlaffaye/ftp v0.2.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/liujitcn/go-utils/http v0.0.6 // indirect
	github.com/liujitcn/go-utils/translator/alibaba v0.0.2 // indirect
	github.com/liujitcn/go-utils/translator/baidu v0.0.3 // indirect
	github.com/liujitcn/go-utils/translator/google v0.0.3 // indirect
	github.com/liujitcn/go-utils/translator/volc v0.0.2 // indirect
	github.com/liujitcn/kratos-kit/broker v0.0.10 // indirect
	github.com/liujitcn/kratos-kit/config v0.0.25 // indirect
	github.com/liujitcn/kratos-kit/database/gorm/driver v0.0.18 // indirect
	github.com/liujitcn/kratos-kit/locker v0.0.15 // indirect
	github.com/liujitcn/kratos-kit/logger v0.0.31 // indirect
	github.com/liujitcn/kratos-kit/oss/s3 v0.0.3 // indirect
	github.com/liujitcn/kratos-kit/queue/redisqueue v0.0.15 // indirect
	github.com/liujitcn/kratos-kit/registry v0.0.22 // indirect
	github.com/liujitcn/kratos-kit/tracer v0.0.16 // indirect
	github.com/liujitcn/kratos-kit/tracing v0.0.11 // indirect
	github.com/liujitcn/kratos-kit/transport v0.0.23 // indirect
	github.com/liujitcn/kratos-kit/transport/keepalive v0.0.12 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/mattn/go-runewidth v0.0.23 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/minio-go/v7 v7.1.0 // indirect
	github.com/modelcontextprotocol/go-sdk v1.7.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mvrilo/go-redoc v0.1.5 // indirect
	github.com/olekukonko/cat v0.0.0-20250911104152-50322a0618f6 // indirect
	github.com/olekukonko/errors v1.2.0 // indirect
	github.com/olekukonko/ll v0.1.6 // indirect
	github.com/olekukonko/tablewriter v1.1.4 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/paulmach/orb v0.11.1 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/redis/go-redis/extra/rediscmd/v9 v9.19.0 // indirect
	github.com/redis/go-redis/extra/redisotel/v9 v9.19.0 // indirect
	github.com/redis/go-redis/v9 v9.19.0 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/swaggest/swgui v1.8.7 // indirect
	github.com/tinylib/msgp v1.6.1 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/vearutop/statigz v1.5.0 // indirect
	github.com/volcengine/volc-sdk-golang v1.0.253 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.63.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.65.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.43.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/exp v0.0.0-20250813145105-42675adae3e6 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
	google.golang.org/api v0.269.0 // indirect
	google.golang.org/genproto v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260818201246-1b0934165a6f // indirect
	gopkg.in/cenkalti/backoff.v1 v1.1.0 // indirect
	gopkg.in/ini.v1 v1.67.1 // indirect
	gorm.io/driver/clickhouse v0.7.0 // indirect
	gorm.io/driver/mysql v1.6.0 // indirect
	gorm.io/driver/postgres v1.6.0 // indirect
	gorm.io/plugin/opentelemetry v0.1.16 // indirect
	gorm.io/plugin/prometheus v0.1.0 // indirect
)
