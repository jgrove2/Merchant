import sqlite3
import os

db_path = os.path.abspath('data/merchant.db')
conn = None

try:
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()

    # Drop existing index if it exists
    print(f"Connecting to database at {db_path}")
    print("Dropping index 'idx_provider_market'...")
    cursor.execute("DROP INDEX IF EXISTS idx_provider_market")

    # Create new unique index
    print("Creating unique index 'idx_provider_market' on (provider_id, external_id)...")
    cursor.execute("CREATE UNIQUE INDEX idx_provider_market ON markets (provider_id, external_id)")

    conn.commit()
    print("Index updated successfully.")

except sqlite3.Error as e:
    print(f"An error occurred: {e}")
finally:
    if conn:
        conn.close()
