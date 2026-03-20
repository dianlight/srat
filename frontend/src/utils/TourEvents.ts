import Emittery from "emittery";

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

/**
 * Payload carried by every guided-tour event.
 * `null` is allowed when a selector does not resolve to a visible element.
 */
export type TourEventPayload = Element | null;

/**
 * Strongly-typed event contract for all tour events in the frontend.
 */
export type TourEventMap = Record<TourEventTypes, TourEventPayload>;

const emitter = new Emittery();
const wrappedListeners = new Map<TourEventTypes, WeakMap<Function, (payload: unknown) => void>>();

const getPayload = (payload: unknown): TourEventPayload => {
	if (
		typeof payload === "object" &&
		payload !== null &&
		"data" in payload
	) {
		return (payload as { data: TourEventPayload }).data;
	}

	return payload as TourEventPayload;
};

const getEventListeners = (event: TourEventTypes) => {
	let eventListeners = wrappedListeners.get(event);
	if (!eventListeners) {
		eventListeners = new WeakMap<Function, (payload: unknown) => void>();
		wrappedListeners.set(event, eventListeners);
	}
	return eventListeners;
};

export const TourEvents = {
	/**
	 * Subscribe to a tour event.
	 * Returns an unsubscribe function that must be called in component cleanup.
	 */
	on: <TEvent extends TourEventTypes>(
		event: TEvent,
		listener: (payload: TourEventMap[TEvent]) => void,
	) => {
		const eventListeners = getEventListeners(event);
		const wrapped = (payload: unknown) => {
			listener(getPayload(payload) as TourEventMap[TEvent]);
		};

		eventListeners.set(listener, wrapped);
		const unsubscribe = emitter.on(event, wrapped);

		return () => {
			unsubscribe();
			eventListeners.delete(listener);
		};
	},

	/**
	 * Subscribe once to a tour event and auto-unsubscribe after first call.
	 */
	once: <TEvent extends TourEventTypes>(
		event: TEvent,
		listener: (payload: TourEventMap[TEvent]) => void,
	) => {
		const unsubscribe = TourEvents.on(event, (payload) => {
			unsubscribe();
			listener(payload);
		});
		return unsubscribe;
	},

	/**
	 * Remove a previously registered listener.
	 */
	off: <TEvent extends TourEventTypes>(
		event: TEvent,
		listener: (payload: TourEventMap[TEvent]) => void,
	) => {
		const eventListeners = getEventListeners(event);
		const wrapped = eventListeners.get(listener);

		if (!wrapped) {
			return;
		}

		emitter.off(event, wrapped);
		eventListeners.delete(listener);
	},

	/**
	 * Emit a tour event safely. Listener errors are caught and logged.
	 */
	emit: async <TEvent extends TourEventTypes>(
		event: TEvent,
		payload: TourEventMap[TEvent],
	) => {
		try {
			await emitter.emit(event, payload);
		} catch (error) {
			console.warn(`[TourEvents] Failed to emit ${event}`, error);
		}
	},

	/**
	 * Clear listeners for a specific event or all events.
	 */
	clearListeners: (event?: TourEventTypes) => {
		if (event) {
			emitter.clearListeners(event);
			wrappedListeners.delete(event);
			return;
		}
		emitter.clearListeners();
		wrappedListeners.clear();
	},
};
