-- Create "todos" table
CREATE TABLE "public"."todos" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "text" character varying(255) NOT NULL,
  "category" character varying(50) NOT NULL,
  "done" boolean NULL DEFAULT false,
  PRIMARY KEY ("id")
);
-- Create index "idx_todos_deleted_at" to table: "todos"
CREATE INDEX "idx_todos_deleted_at" ON "public"."todos" ("deleted_at");
