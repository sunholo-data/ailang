class User:
    def __init__(self, name, age, email):
        self.name = name
        self.age = age
        self.email = email

def validateEmail(email):
    return "@" in email
