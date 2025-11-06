import json
from data import User

def saveUser(user, filename):
    data = {
        "name": user.name,
        "age": user.age,
        "email": user.email
    }
    with open(filename, "w") as f:
        json.dump(data, f)

def loadUser(filename):
    with open(filename, "r") as f:
        data = json.load(f)
    return User(data["name"], data["age"], data["email"])
