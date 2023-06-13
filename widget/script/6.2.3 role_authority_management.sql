## root
DROP USER IF EXISTS tidb_user;
CREATE TABLE ${TABLE_NAME} (id INT PRIMARY KEY);
INSERT INTO ${TABLE_NAME} VALUES (1), (2), (3);
DROP ROLE IF EXISTS tidb_role;
CREATE ROLE tidb_role;
GRANT SELECT ON ${TABLE_NAME} TO tidb_role;
CREATE USER tidb_user IDENTIFIED BY 'tidb_password';
GRANT tidb_role TO tidb_user;
## -
## tidb_user
SELECT * FROM ${TABLE_NAME};
INSERT INTO ${TABLE_NAME} VALUES (4), (5);
## -
## root
REVOKE SELECT ON *.* FROM 'tidb_role';
DROP ROLE 'tidb_role';
DROP USER 'tidb_user';