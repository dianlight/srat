import type { User } from "../../store/sratApi";

export interface UsersProps extends User {
	doCreate?: boolean;
	"password-repeat"?: string;
}
