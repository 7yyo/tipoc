## root
SELECT CURRENT_USER;
DROP USER IF EXISTS tidb_user;
CREATE USER 'tidb_user'@'%' IDENTIFIED BY 'tidb_password' FAILED_LOGIN_ATTEMPTS 2 PASSWORD_LOCK_TIME 3;
## -
## tidb_user
## -
## tidb_user
## -
## root
DROP USER IF EXISTS tidb_user;