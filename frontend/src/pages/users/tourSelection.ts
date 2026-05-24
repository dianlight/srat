import type { User } from "../../store/sratApi";

function isValidUser(user: User | null | undefined): user is User {
  return Boolean(user && typeof user === "object");
}

export function getTourTargetUser(
  users?: ReadonlyArray<User | null | undefined> | null,
): User | undefined {
  if (!Array.isArray(users) || users.length === 0) return undefined;

  const validUsers = users.filter(isValidUser);
  return validUsers.find((user) => !user.is_admin) ?? validUsers[0];
}
