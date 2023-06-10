## root
SET GLOBAL validate_password.enable = ON;
SET PASSWORD FOR root@'%' = '12345';
SET GLOBAL validate_password.enable = OFF;