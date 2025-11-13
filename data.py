class User:
    def __init__(self, name: str, age: int, email: str) -> None:
        self.name = name
        self.age = age
        self.email = email


def validateEmail(email: str) -> bool:
    return "@" in email
