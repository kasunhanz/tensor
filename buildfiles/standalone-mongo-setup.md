### Setup MongoDB users

Create database admin

```
use admin;
db.createUser(
  {
    user: "admin",
    pwd: "password",
    roles: [ { role: "root", db: "admin" } ]
  }
);
```

Create hilbertspace database admin

```
use admin;
db.createUser(
  {
    user: "hilbert",
    pwd: "hilbert",
    db: "hilbertspace",
    roles: [ { role: "dbOwner", db: "hilbertspace" } ]
  }
);
```