USE poc;
CREATE TABLE employees
(
    id     INT AUTO_INCREMENT PRIMARY KEY,
    name   VARCHAR(255) NOT NULL,
    age    INT,
    salary DECIMAL(10, 2)
);

DELIMITER //

CREATE PROCEDURE InsertEmployee(
    IN empName VARCHAR(255),
    IN empAge INT,
    IN empSalary DECIMAL(10, 2)
)
BEGIN
    INSERT INTO employees (name, age, salary)
    VALUES (empName, empAge, empSalary);
END //

DELIMITER ;