CREATE TABLE ${TABLE_NAME}
(
    id     INT PRIMARY KEY,
    name   VARCHAR(50),
    course VARCHAR(50)
);
INSERT INTO ${TABLE_NAME} (id, name, course)
VALUES (1, 'Alice', 'Math'),
       (2, 'Bob', 'Science'),
       (3, 'Charlie', 'Math'),
       (4, 'David', 'History'),
       (5, 'Emma', 'Science');
SELECT course,
       GROUP_CONCAT(name ORDER BY name SEPARATOR ', ') AS student_list
FROM ${TABLE_NAME}
GROUP BY course;