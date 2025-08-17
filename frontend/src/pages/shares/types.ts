import type { SharedResource } from "../../store/sratApi";

export interface ShareEditProps extends SharedResource {
	org_name: string;
}

export enum CasingStyle {
	UPPERCASE = "UPPERCASE",
	LOWERCASE = "lowercase",
	CAMELCASE = "camelCase",
	KEBABCASE = "kebab-case",
}
