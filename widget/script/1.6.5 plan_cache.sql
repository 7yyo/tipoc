CREATE TABLE ${TABLE_NAME}_s (id INT PRIMARY KEY);
INSERT INTO ${TABLE_NAME}_s VALUE (1);
set tidb_enable_non_prepared_plan_cache = on;
SELECT * FROM ${TABLE_NAME}_s;
SELECT @@last_plan_from_cache;
SELECT * FROM ${TABLE_NAME}_s;
select @@last_plan_from_cache;

CREATE TABLE ${TABLE_NAME}_p (id INT PRIMARY KEY);
INSERT INTO ${TABLE_NAME}_p VALUE (1);
PREPARE ps FROM 'SELECT * FROM ${TABLE_NAME}_p WHERE id = ?';
SET @a = 1;
EXECUTE ps USING @a;
SELECT @@last_plan_from_cache;
EXECUTE ps USING @a;
SELECT @@last_plan_from_cache;