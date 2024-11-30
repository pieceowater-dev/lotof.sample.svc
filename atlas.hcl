data "external_schema" "postgres" {
  program = ["go", "run", "./cmd/server/db/pg/pg-migrate.go"]
}

env "postgres" {
  src = data.external_schema.postgres.url
  dev = "docker://postgres/16/dev"
  migration {
    dir = "file://cmd/server/db/pg/migrations"
  }

  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}