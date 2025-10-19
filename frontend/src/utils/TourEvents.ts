import Emittery from "emittery";

const emitter = new Emittery();

export const TourEvents = {
	on: (event: TourEventTypes, listener: (...args: unknown[]) => void) => {
		emitter.on(event, listener);
	},
	off: (event: TourEventTypes, listener: (...args: unknown[]) => void) => {
		emitter.off(event, listener);
	},
	emit: (event: TourEventTypes, ...args: unknown[]) => {
		emitter.emit(event, args);
	},
};

export enum TourEventTypes {
	// Dashboard steps
	DASHBOARD_STEP_2 = "tour:dashboard:step2",
	DASHBOARD_STEP_3 = "tour:dashboard:step3",
	DASHBOARD_STEP_4 = "tour:dashboard:step4",
	DASHBOARD_STEP_5 = "tour:dashboard:step5",
	DASHBOARD_STEP_6 = "tour:dashboard:step6",
	DASHBOARD_STEP_7 = "tour:dashboard:step7",
	DASHBOARD_STEP_8 = "tour:dashboard:step8",

	// Shares steps
	//SHARES_STEP_2 = "tour:shares:step2",
	SHARES_STEP_3 = "tour:shares:step3",
	SHARES_STEP_4 = "tour:shares:step4",

	// Volumes steps
	VOLUMES_STEP_3 = "tour:volumes:step3",
	VOLUMES_STEP_4 = "tour:volumes:step4",
	VOLUMES_STEP_5 = "tour:volumes:step5",

	// Settings steps
	SETTINGS_STEP_3 = "tour:settings:step3",
	SETTINGS_STEP_5 = "tour:settings:step5",
	SETTINGS_STEP_8 = "tour:settings:step8",

	// Users steps
	USERS_STEP_3 = "tour:users:step3",
}
