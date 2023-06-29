## root
DROP USER IF EXISTS tidb_user;
CREATE USER tidb_user IDENTIFIED by 'tidb_password';
SELECT host, user, plugin FROM mysql.user WHERE USER = 'tidb_user';
DROP USER tidb_user;