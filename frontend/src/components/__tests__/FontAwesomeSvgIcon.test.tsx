import React from "react";
import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithTestStore } from "/test/testing";

type FaIcon = {
    icon: [number, number, unknown, unknown, string | string[]];
};

describe("FontAwesomeSvgIcon Component", () => {
    async function renderIcon(icon: FaIcon, props?: Record<string, unknown>) {
        const { FontAwesomeSvgIcon } = await import("../FontAwesomeSvgIcon");
        return renderWithTestStore(
            React.createElement(FontAwesomeSvgIcon, { icon, ...props })
        );
    }

    it("renders with single path icon data", async () => {
        const singlePathIcon: FaIcon = {
            icon: [16, 16, [], "f000", "M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8 8-3.58 8-8-3.58-8-8-8z"]
        };

        await renderIcon(singlePathIcon);

        const svgElement = screen.getByTestId("fontawesome-svg-icon");
        expect(svgElement).toBeTruthy();
        expect(svgElement.tagName).toBe('svg');
        expect(svgElement.getAttribute('viewBox')).toBe('0 0 16 16');

        const pathElements = within(svgElement).getAllByTestId("fontawesome-icon-path");
        expect(pathElements.length).toBe(1);
        expect(pathElements[0]?.getAttribute('d')).toBe('M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8 8-3.58 8-8-3.58-8-8-8z');
    });

    it("renders with multi-path icon data (duotone)", async () => {
        const multiPathIcon: FaIcon = {
            icon: [
                24,
                24,
                [],
                "f001",
                [
                    "M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8 8-3.58 8-8-3.58-8-8-8z",
                    "M16 8c0-4.42-3.58-8-8-8v16c4.42 0 8-3.58 8-8z"
                ]
            ]
        };

        await renderIcon(multiPathIcon);

        const svgElement = screen.getByTestId("fontawesome-svg-icon");
        expect(svgElement).toBeTruthy();
        expect(svgElement.tagName).toBe('svg');
        expect(svgElement.getAttribute('viewBox')).toBe('0 0 24 24');

        const pathElements = within(svgElement).getAllByTestId("fontawesome-icon-path");
        expect(pathElements.length).toBe(2);
        expect(pathElements[0]?.style.opacity).toBe('0.4');
        expect(pathElements[0]?.getAttribute('d')).toBe('M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8 8-3.58 8-8-3.58-8-8-8z');
        expect(pathElements[1]?.style.opacity).toBe('1');
        expect(pathElements[1]?.getAttribute('d')).toBe('M16 8c0-4.42-3.58-8-8-8v16c4.42 0 8-3.58 8-8z');
    });

    it("handles different icon dimensions correctly", async () => {
        const customSizeIcon: FaIcon = {
            icon: [32, 20, [], "f002", "M0 0h32v20H0z"]
        };

        await renderIcon(customSizeIcon);

        const svgElement = screen.getByTestId("fontawesome-svg-icon");
        expect(svgElement).toBeTruthy();
        expect(svgElement.tagName).toBe('svg');
        expect(svgElement.getAttribute('viewBox')).toBe('0 0 32 20');
    });

    it("forwards ref correctly", async () => {
        const singlePathIcon: FaIcon = {
            icon: [16, 16, [], "f000", "M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8 8-3.58 8-8-3.58-8-8-8z"]
        };

        const ref = React.createRef<SVGSVGElement>();

        await renderIcon(singlePathIcon, { ref });

        expect(ref.current).toBeTruthy();
        expect(ref.current?.tagName).toBe('svg');
    });

    it("handles empty multi-path array", async () => {
        const emptyMultiPathIcon: FaIcon = {
            icon: [16, 16, [], "f003", []]
        };

        await renderIcon(emptyMultiPathIcon);

        const svgElement = screen.getByTestId("fontawesome-svg-icon");
        expect(svgElement).toBeTruthy();
        expect(svgElement.tagName).toBe('svg');

        expect(within(svgElement).queryAllByTestId("fontawesome-icon-path").length).toBe(0);
    });

    it("handles complex multi-path duotone with more than 2 paths", async () => {
        const complexMultiPathIcon: FaIcon = {
            icon: [
                16,
                16,
                [],
                "f004",
                [
                    "M0 0h4v4H0z",
                    "M4 0h4v4H4z",
                    "M8 0h4v4H8z"
                ]
            ]
        };

        await renderIcon(complexMultiPathIcon);

        const svgElement = screen.getByTestId("fontawesome-svg-icon");
        const pathElements = within(svgElement).getAllByTestId("fontawesome-icon-path");
        expect(pathElements.length).toBe(3);
        expect(pathElements[0]?.style.opacity).toBe('0.4');
        expect(pathElements[1]?.style.opacity).toBe('1');
        expect(pathElements[2]?.style.opacity).toBe('1');
    });

    it("defaults to medium size when no fontSize prop is passed", async () => {
        const singlePathIcon: FaIcon = {
            icon: [16, 16, [], "f000", "M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8 8-3.58 8-8-3.58-8-8-8z"]
        };

        await renderIcon(singlePathIcon);

        const svgElement = screen.getByTestId("fontawesome-svg-icon");
        expect(svgElement.getAttribute('class')).toContain('MuiSvgIcon-fontSizeMedium');
    });

    it("applies the small size class when fontSize='small' is passed", async () => {
        const singlePathIcon: FaIcon = {
            icon: [16, 16, [], "f000", "M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8-3.58 8-8-8-3.58-8-8-8z"]
        };

        await renderIcon(singlePathIcon, { fontSize: "small" });

        const svgElement = screen.getByTestId("fontawesome-svg-icon");
        expect(svgElement.getAttribute('class')).toContain('MuiSvgIcon-fontSizeSmall');
        expect(svgElement.getAttribute('class')).not.toContain('MuiSvgIcon-fontSizeMedium');
    });
});
