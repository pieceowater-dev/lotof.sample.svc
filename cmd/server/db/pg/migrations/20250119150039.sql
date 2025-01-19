-- Create "somethings" table
CREATE TABLE "public"."somethings" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "some_enum" bigint NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_somethings_deleted_at" to table: "somethings"
CREATE INDEX "idx_somethings_deleted_at" ON "public"."somethings" ("deleted_at");
