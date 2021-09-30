module github.com/crawlab-team/plugin-dependency

go 1.15

replace (
	github.com/crawlab-team/crawlab-core => /Users/marvzhang/projects/crawlab-team/crawlab-core
	github.com/crawlab-team/crawlab-grpc => /Users/marvzhang/projects/crawlab-team/crawlab-grpc
	github.com/crawlab-team/crawlab-plugin => /Users/marvzhang/projects/crawlab-team/crawlab-plugin
)

require (
	github.com/PuerkitoBio/goquery v1.7.1
	github.com/apex/log v1.9.0
	github.com/crawlab-team/crawlab-core v0.6.0-beta.20210802.1344
	github.com/crawlab-team/crawlab-db v0.1.1
	github.com/crawlab-team/crawlab-grpc v0.6.0-beta.20210922.1610
	github.com/crawlab-team/crawlab-plugin v0.0.0-20210604093326-57f35f02daf5
	github.com/crawlab-team/go-trace v0.1.0
	github.com/gin-gonic/gin v1.6.3
	github.com/imroc/req v0.3.0
	go.mongodb.org/mongo-driver v1.4.5
	go.uber.org/dig v1.10.0
)
