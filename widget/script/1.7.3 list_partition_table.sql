SET @@session.tidb_enable_list_partition = ON;
CREATE TABLE ${TABLE_NAME} (id INT PRIMARY KEY)
    PARTITION BY LIST (id) (
        PARTITION p0 VALUES IN (1,2,3,4,5),
        PARTITION p1 VALUES IN (6,7,8,9,0));
INSERT INTO ${TABLE_NAME} VALUES (1),(2),(3),(4),(5),(6),(7),(8),(9),(0);
SELECT * FROM ${TABLE_NAME} PARTITION (p1);