# EcoCampus Passport Backend
The EcoCampus backend is a REST API service made using golang and utilizing
postgresql for the database.

# Dependencies
- Docker
- Golang >= 1.24.1

# Running in production

# Running in test
```
$ docker compose up postgres-test -d
$ go run main.go
```

# Contributing
Ecocampus Passport is open to contributions and if you're interested in working on this project
please contact CMU-Q facilities department. We welcome pull requests but to ensure it gets
merged we recommend you to first speak to facilities and get permission on whatever you're planning
on working on. PRs which fix bugs are an exception to this. 

When contributing please follow [Commit Message Guidlines](https://gist.github.com/robertpainsi/b632364184e70900af4ab688decf6f53).


# Security disclosure
If you find a security bug please disclose it immediately to helpcenter@qatar.cmu.edu and talhah@cmu.edu along
with how to reproduce it. You are not allowed to disclose the bug to other parties
until we give you clearance.

# License 
This project is licensed under the [MIT License](LICENSE).
