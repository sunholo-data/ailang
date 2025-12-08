---
description: Validate AILANG code syntax using existing scripts.
---

# Validate AILANG Code

This workflow validates AILANG code syntax using the `validate_code.sh` script from the `use-ailang` skill.

## Steps

1.  **Check AILANG Version**
    Verify the active AILANG version to ensure we are validating against the correct standard.
    ```bash
    .claude/skills/use-ailang/scripts/check_version.sh
    ```

2.  **Validate Code File**
    Run the validation script on the target file.
    Replace `[FILE_PATH]` with the actual path to the `.ail` file.
    ```bash
    .claude/skills/use-ailang/scripts/validate_code.sh [FILE_PATH]
    ```

    **Expected Output:**
    - `✓ Type check passed` if valid.
    - Error messages if invalid.

## Notes
- This workflow relies on `ailang check` under the hood.
- Ensure `ailang` is in your PATH (run `make install` if not).
