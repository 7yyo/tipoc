SHOW CONFIG WHERE type = 'pd' AND name = 'log.level';
SET CONFIG pd `log.level`='warn';
SHOW CONFIG WHERE type = 'pd' AND name = 'log.level';
SET CONFIG pd `log.level`='info';