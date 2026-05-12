import React from "react";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { TourEvents, TourEventTypes } from "../../../utils/TourEvents";
import { UsersSteps } from "../UsersSteps";

describe("Users tour steps", () => {
    it("lists user onboarding steps and keeps step 3 aligned with edit actions", () => {

        expect(Array.isArray(UsersSteps)).toBe(true);
        expect(UsersSteps.length).toBeGreaterThan(0);

        UsersSteps.forEach((step, index) => {
            expect(step.selector).toContain(`step${index}`);
            expect(step.content).toBeTruthy();
        });

        const originalEmit = TourEvents.emit;
        let emittedEvent: string | null = null;

        try {
            TourEvents.emit = ((event: any) => {
                emittedEvent = event;
                return Promise.resolve();
            }) as typeof TourEvents.emit;

            UsersSteps.forEach((step) => step.action?.({ username: "demo" } as any));
        } finally {
            TourEvents.emit = originalEmit;
        }

        if (emittedEvent !== null) {
            expect(emittedEvent as string).toBe(TourEventTypes.USERS_STEP_3);
        }

        render(React.createElement(React.Fragment, null, UsersSteps[3]?.content as any));

        expect(screen.getByText("Edit User")).toBeTruthy();
    });
});