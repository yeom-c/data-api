## [Data API](http://data-api.quasar-gamestudio.ga)

### 소개

> Data API 는 데이터툴 서비스에 필요한 API 를 제공합니다.

- [Go](https://go.dev)
- [Fiber](https://gofiber.io)
- [Golang Migrate](https://github.com/golang-migrate/migrate)
- [XORM](https://xorm.io)
- [Air](https://github.com/cosmtrek/air)

### 구조

```
- app
: 서버 실행을 위한 fiber app instance, config 환경변수
- auth_token
: 클라이언트 인증 토큰을 생성하기 위한 maker
- database
: 데이터베이스 사용을 위한 xorm engine, model, migration 파일
- handler
: request 처리를 위한 fiber 핸들러 모음
- middleware
: fiber 미들웨어, 커스텀 미들웨어
- route
: api route
- util
: 공통으로 사용되는 함수
```

### 데이터베이스 마이그레이션

- migrate CLI 설치: [migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)

```
- 마이그레이션 생성: make migrate-create
- 마이그레이션 적용: make migrate-up
- 마이그레이션 롤백: make migrate-down
```

### 시작하기

```bash
# 프로젝트 클론
git clone git@github.com:yeom-c/data-api.git

# 프로젝트 폴더로 이동
cd quasar-data-api

# .env, Makefile 파일 복사 후 열어서 자신의 환경에 맞게 변수 설정
cp .env.example .env
cp Makefile.example Makefile

# 의존성 모듈 설치
go mod download

# 실행
go run application.go
# or air 가 설치되어 있다면 air 로 실행
air
```

### 배포

- main 브랜치에 push 할 경우 서비스 환경에 배포
