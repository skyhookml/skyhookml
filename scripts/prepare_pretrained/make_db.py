import json
import sqlite3
import sys

db_fname = sys.argv[1]
label = sys.argv[2]

conn = sqlite3.connect(db_fname)
c = conn.cursor()
c.execute('''
CREATE TABLE items (
	-- item key
	k TEXT PRIMARY KEY,
	ext TEXT,
	format TEXT,
	metadata TEXT,
	-- set if LoadData call should go through non-default method, else NULL
	provider TEXT,
	provider_info TEXT
)
''')
c.execute('''
CREATE TABLE datasets (
	id INTEGER PRIMARY KEY ASC,
	name TEXT,
	-- 'data' or 'computed'
	type TEXT,
	data_type TEXT,
	-- only set if computed
	hash TEXT
)
''')
c.execute("INSERT INTO datasets (id, name, type, data_type) VALUES (1, ?, 'data', 'file')", (label,))
c.execute("INSERT INTO items (k, ext, format, metadata) VALUES ('model', 'pt', '', ?)", (json.dumps({'Filename': 'model.pt'}),))
conn.commit()
conn.close()
