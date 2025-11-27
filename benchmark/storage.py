import json
from pathlib import Path
from data import User
from typing import Any, Dict

def saveUser(user: User, filename: str) -> None:
    data: Dict[str, Any] = {
        "name": user.name,
        "age": user.age,
        "email": user.email
    }
    path = Path(filename)
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as f:
        json.dump(data, f)

def loadUser(filename: str) -> User:
    path = Path(filename)
    with path.open("r", encoding="utf-8") as f:
        data = json.load(f)
    return User(data["name"], data["age"], data["email"])
