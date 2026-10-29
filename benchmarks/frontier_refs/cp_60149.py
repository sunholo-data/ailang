def parse_digit(s):
    if len(s) == 1:
        if s.isdigit():
            return int(s)
        else:
            return ord(s) - ord('A') + 10
    else:
        return int(s)

def digit_to_char(d):
    if d < 10:
        return str(d)
    else:
        return chr(ord('A') + d - 10)

def solve(N, K, forbidden):
    all_digits = set(range(N))
    forbidden_set = set(forbidden)
    allowed = sorted(all_digits - forbidden_set)

    m = len(allowed)
    non_zero_allowed = [d for d in allowed if d != 0]
    n_nz = len(non_zero_allowed)

    # Find which "digit length" K falls into
    count = 0
    digits = 1
    while count + n_nz * (m ** (digits - 1)) < K:
        count += n_nz * (m ** (digits - 1))
        digits += 1

    k_in_group = K - count

    result = []

    # First digit (cannot be 0)
    first_digit_index = (k_in_group - 1) // (m ** (digits - 1))
    result.append(non_zero_allowed[first_digit_index])

    remaining = (k_in_group - 1) % (m ** (digits - 1))

    # Remaining digits
    for i in range(digits - 2, -1, -1):
        digit_index = remaining // (m ** i)
        result.append(allowed[digit_index])
        remaining %= m ** i

    return ''.join(digit_to_char(d) for d in result)

T = int(input())
for _ in range(T):
    N, K, M = map(int, input().split())
    if M > 0:
        forbidden_strs = input().split()
        forbidden = [parse_digit(s) for s in forbidden_strs]
    else:
        input()
        forbidden = []

    print(solve(N, K, forbidden))
