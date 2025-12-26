import sqlite3
import os

db_path = 'data/merchant.db'

if not os.path.exists(db_path):
    print(f"Error: Database not found at {db_path}")
else:
    try:
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()
        
        # Get CREATE TABLE statement
        cursor.execute("SELECT sql FROM sqlite_master WHERE type='table' AND name='markets'")
        table_info = cursor.fetchone()
        
        if table_info:
            print("--- CREATE TABLE Statement ---")
            print(table_info[0])
        else:
            print("Table 'markets' not found.")

        # Get Indices
        print("\n--- Indices ---")
        cursor.execute("SELECT name, sql FROM sqlite_master WHERE type='index' AND tbl_name='markets'")
        indices = cursor.fetchall()
        if indices:
            for name, sql in indices:
                print(f"Index: {name}")
                print(f"SQL: {sql}")
        else:
            print("No indices found.")
            
        conn.close()
    except sqlite3.Error as e:
        print(f"SQLite error: {e}")
