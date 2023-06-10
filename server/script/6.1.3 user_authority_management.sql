## root
SELECT CURRENT_USER;
DROP USER IF EXISTS 'tidb_user'@'%';
CREATE TABLE ${TABLE_NAME} (id INT PRIMARY KEY );
INSERT INTO ${TABLE_NAME} VALUES (1), (2), (3);
CREATE USER 'tidb_user'@'%' IDENTIFIED BY 'tidb_password';
GRANT SELECT ON poc.* TO 'tidb_user'@'%';
## -
## tidb_user
SELECT CURRENT_USER;
SELECT * FROM ${TABLE_NAME};
INSERT INTO ${TABLE_NAME} VALUES (4), (5);
## -
## root
SELECT CURRENT_USER;
REVOKE ALL PRIVILEGES ON poc.* FROM 'tidb_user'@'%';
DROP USER 'tidb_user'@'%';
