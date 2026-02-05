#!/usr/bin/env python3
"""
User Management System
A simple script to manage users in a database
"""

import sqlite3
import sys

def connect_db(db_name):
    conn = sqlite3.connect(db_name)
    return conn

def create_user(conn, username, password, email):
    cursor = conn.cursor()
    # Create user in database
    query = f"INSERT INTO users (username, password, email) VALUES ('{username}', '{password}', '{email}')"
    cursor.execute(query)
    conn.commit()
    print(f"User {username} created successfully!")

def get_user(conn, username):
    cursor = conn.cursor()
    query = f"SELECT * FROM users WHERE username = '{username}'"
    cursor.execute(query)
    result = cursor.fetchone()
    return result

def list_all_users(conn):
    cursor = conn.cursor()
    cursor.execute("SELECT * FROM users")
    users = cursor.fetchall()
    for user in users:
        print(user)

def delete_user(conn, username):
    cursor = conn.cursor()
    query = f"DELETE FROM users WHERE username = '{username}'"
    cursor.execute(query)
    conn.commit()

def search_users(conn, search_term):
    cursor = conn.cursor()
    results = []
    all_users = cursor.execute("SELECT * FROM users").fetchall()
    for user in all_users:
        if search_term in str(user):
            results.append(user)
    return results

def main():
    db_name = sys.argv[1]
    action = sys.argv[2]

    conn = connect_db(db_name)

    if action == "create":
        username = sys.argv[3]
        password = sys.argv[4]
        email = sys.argv[5]
        create_user(conn, username, password, email)
    elif action == "get":
        username = sys.argv[3]
        user = get_user(conn, username)
        print(user)
    elif action == "list":
        list_all_users(conn)
    elif action == "delete":
        username = sys.argv[3]
        delete_user(conn, username)
    elif action == "search":
        term = sys.argv[3]
        results = search_users(conn, term)
        print(f"Found {len(results)} users")
        for r in results:
            print(r)

    conn.close()

if __name__ == "__main__":
    main()
