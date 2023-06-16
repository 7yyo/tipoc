CREATE TABLE ${TABLE_NAME}(id INT, modified DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, custinfo JSON,
    INDEX zips ((CAST(custinfo -> '$.zipcode' AS UNSIGNED ARRAY)))
);
INSERT INTO ${TABLE_NAME} VALUES
         (NULL, NOW(), '{"user":"Jack","user_id":37,"zipcode":[94582,94536]}'),
         (NULL, NOW(), '{"user":"Jill","user_id":22,"zipcode":[94568,94507,94582]}'),
         (NULL, NOW(), '{"user":"Bob","user_id":31,"zipcode":[94477,94507]}'),
         (NULL, NOW(), '{"user":"Mary","user_id":72,"zipcode":[94536]}'),
         (NULL, NOW(), '{"user":"Ted","user_id":56,"zipcode":[94507,94582]}');
ANALYZE TABLE ${TABLE_NAME};
SELECT * FROM ${TABLE_NAME} WHERE 94507 MEMBER OF (custinfo->'$.zipcode');
EXPLAIN SELECT * FROM ${TABLE_NAME} WHERE 94507 MEMBER OF (custinfo->'$.zipcode');