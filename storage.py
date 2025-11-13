import json
from pathlib import Path
from data import User


def saveUser(user: User, filename: str) -> None:
    data = {
        "name": user.name,
        "age": user.age,
        "email": user.email,
    }
    path = Path(filename)
    path.write_text(json.dumps(data), encoding="utf-8")


def loadUser(filename: str) -> User:
    path = Path(filename)
    content = path.read_text(encoding="utf-8")
    data = json.loads(content)
    return User(data["name"], data["age"], data["email"])
