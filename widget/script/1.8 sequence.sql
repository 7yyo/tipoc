CREATE SEQUENCE ${TABLE_NAME}_1;
SELECT NEXTVAL(${TABLE_NAME}_1);
SELECT SETVAL(${TABLE_NAME}_1, 10);
CREATE SEQUENCE ${TABLE_NAME}_2 INCREMENT = 2;
SELECT NEXTVAL(${TABLE_NAME}_2);
SELECT NEXTVAL(${TABLE_NAME}_2);
CREATE TABLE ${TABLE_NAME}_tbl (id INT DEFAULT NEXT VALUE FOR ${TABLE_NAME}_2);
INSERT INTO ${TABLE_NAME}_tbl VALUES ();
SELECT * FROM ${TABLE_NAME}_tbl;
