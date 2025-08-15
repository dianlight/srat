import Emittery from 'emittery';

const emitter = new Emittery();

export const TourEvents = {
    on: (event: TourEventTypes, listener: (...args: any[]) => void) => {
        emitter.on(event, listener);
    },
    off: (event: TourEventTypes, listener: (...args: any[]) => void) => {
        emitter.off(event, listener);
    },
    emit: (event: TourEventTypes, ...args: any[]) => {
        emitter.emit(event, args);
    },
};

export enum TourEventTypes {
    DASHBOARD_STEP_2 = "tour:dashboard:step2",
    DASHBOARD_STEP_3 = "tour:dashboard:step3",
    DASHBOARD_STEP_4 = "tour:dashboard:step4",
    DASHBOARD_STEP_5 = "tour:dashboard:step5",
    DASHBOARD_STEP_6 = "tour:dashboard:step6",
    DASHBOARD_STEP_7 = "tour:dashboard:step7",
    DASHBOARD_STEP_8 = "tour:dashboard:step8",
}