## Tensor

Centralized infrastructure management REST API, based on ansible, provides role-based access control, job scheduling, inventory management.
Currently, the REST API supports the only ansible. Our expectation is to support other CI-CD automation tools like Chef,Puppet in the near future.

**Use Cases**

- Configuration Management
- Provisioning
- Code Deployments
- Continuous Integration & Continuous Delivery
- Security & Compliance
- Orchestration

updating....

### Installation instructions


```
docker-compose up
go get -u github.com/jteeuwen/go-bindata/...
go get github.com/mitchellh/gox
go get github.com/cespare/reflex
```

```
github.com/gin-gonic/gin
github.com/go-sql-driver/mysql
github.com/google/go-github/github
github.com/gorilla/securecookie
github.com/gorilla/websocket
github.com/masterminds/squirrel
github.com/russross/blackfriday
golang.org/x/crypto/bcrypt
gopkg.in/gorp.v1
```

**Build dependacies**

sudo apt-get install g++-multilib