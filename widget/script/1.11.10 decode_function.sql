CREATE TABLE ${TABLE_NAME}
(
    id     INT PRIMARY KEY,
    name   VARCHAR(50),
    dept   VARCHAR(10),
    salary INT
);

INSERT INTO ${TABLE_NAME} (id, name, dept, salary)
VALUES (1, 'Alice', 'HR', 3000),
       (2, 'Bob', 'IT', 4000),
       (3, 'Charlie', 'HR', 3500),
       (4, 'David', 'IT', 4200);

SELECT name,
       dept,
       CASE
           WHEN salary = 3000 THEN 'Low'
           WHEN salary = 3500 THEN 'Medium'
           WHEN salary = 4000 THEN 'High'
           ELSE 'Other'
           END AS salary_level
FROM ${TABLE_NAME};
