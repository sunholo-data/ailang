from data import User, validateEmail
from storage import saveUser, loadUser


def main() -> None:
    user = User("Alice", 30, "alice@example.com")
    valid = validateEmail(user.email)
    print(f"Email valid: {valid}")

    saveUser(user, "user.json")
    loaded = loadUser("user.json")
    print(f"Loaded: {loaded.name}, age {loaded.age}")


if __name__ == "__main__":
    main()
