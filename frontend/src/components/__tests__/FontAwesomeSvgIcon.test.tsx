import "../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

describe("FontAwesomeSvgIcon Component", () => {
    beforeEach(() => {
        // Clear the DOM before each test
        document.body.innerHTML = '';
    });

    it("renders with single path icon data", async () => {
        // Dynamic imports for React components
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { FontAwesomeSvgIcon } = await import("../FontAwesomeSvgIcon");

        const singlePathIcon = {
            icon: [16, 16, [], "f000", "M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8 8-3.58 8-8-3.58-8-8-8z"]
        };

        const { container } = render(
            React.createElement(FontAwesomeSvgIcon, { icon: singlePathIcon })
        );

        // Check that an SVG element is rendered (SVG has no semantic role, must use container)
        const svgElement = container.firstChild as SVGSVGElement;
        expect(svgElement).toBeTruthy();
        expect(svgElement?.tagName).toBe('svg');

        // Check that the viewBox is set correctly
        expect(svgElement?.getAttribute('viewBox')).toBe('0 0 16 16');

        // Check that a single path element is rendered
        const pathElements = svgElement?.getElementsByTagName('path');
        expect(pathElements?.length).toBe(1);
        expect(pathElements?.[0]?.getAttribute('d')).toBe('M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8 8-3.58 8-8-3.58-8-8-8z');
    });

    it("renders with multi-path icon data (duotone)", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { FontAwesomeSvgIcon } = await import("../FontAwesomeSvgIcon");

        const multiPathIcon = {
            icon: [
                24,
                24,
                [],
                "f001",
                [
                    "M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8 8-3.58 8-8-3.58-8-8-8z", // Secondary path (40% opacity)
                    "M16 8c0-4.42-3.58-8-8-8v16c4.42 0 8-3.58 8-8z" // Primary path (100% opacity)
                ]
            ]
        };

        const { container } = render(
            React.createElement(FontAwesomeSvgIcon, { icon: multiPathIcon })
        );

        const svgElement = container.firstChild as SVGSVGElement;
        expect(svgElement).toBeTruthy();
        expect(svgElement?.tagName).toBe('svg');

        // Check that the viewBox is set correctly for 24x24 icon
        expect(svgElement?.getAttribute('viewBox')).toBe('0 0 24 24');

        // Check that both path elements are rendered
        const pathElements = svgElement?.getElementsByTagName('path');
        expect(pathElements?.length).toBe(2);

        // Check that the first path has 40% opacity (secondary/faded element)
        expect(pathElements?.[0]?.style.opacity).toBe('0.4');
        expect(pathElements?.[0]?.getAttribute('d')).toBe('M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8 8-3.58 8-8-3.58-8-8-8z');

        // Check that the second path has 100% opacity (primary element)
        expect(pathElements?.[1]?.style.opacity).toBe('1');
        expect(pathElements?.[1]?.getAttribute('d')).toBe('M16 8c0-4.42-3.58-8-8-8v16c4.42 0 8-3.58 8-8z');
    });

    it("handles different icon dimensions correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { FontAwesomeSvgIcon } = await import("../FontAwesomeSvgIcon");

        const customSizeIcon = {
            icon: [32, 20, [], "f002", "M0 0h32v20H0z"]
        };

        const { container } = render(
            React.createElement(FontAwesomeSvgIcon, { icon: customSizeIcon })
        );

        const svgElement = container.firstChild as SVGSVGElement;
        expect(svgElement).toBeTruthy();
        expect(svgElement?.tagName).toBe('svg');

        // Check that the viewBox respects custom dimensions
        expect(svgElement?.getAttribute('viewBox')).toBe('0 0 32 20');
    });

    it("forwards ref correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { FontAwesomeSvgIcon } = await import("../FontAwesomeSvgIcon");

        const singlePathIcon = {
            icon: [16, 16, [], "f000", "M8 0C3.58 0 0 3.58 0 8s3.58 8 8 8 8-3.58 8-8-3.58-8-8-8z"]
        };

        const ref = React.createRef<SVGSVGElement>();

        const { container } = render(
            React.createElement(FontAwesomeSvgIcon, { icon: singlePathIcon, ref })
        );

        // Check that the ref is properly forwarded to the SVG element
        expect(ref.current).toBeTruthy();
        expect(ref.current?.tagName).toBe('svg');
    });

    it("handles empty multi-path array", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { FontAwesomeSvgIcon } = await import("../FontAwesomeSvgIcon");

        const emptyMultiPathIcon = {
            icon: [16, 16, [], "f003", []]
        };

        const { container } = render(
            React.createElement(FontAwesomeSvgIcon, { icon: emptyMultiPathIcon })
        );

        const svgElement = container.firstChild as SVGSVGElement;
        expect(svgElement).toBeTruthy();
        expect(svgElement?.tagName).toBe('svg');

        // Should have no path elements when array is empty
        const pathElements = svgElement?.getElementsByTagName('path');
        expect(pathElements?.length).toBe(0);
    });

    it("handles complex multi-path duotone with more than 2 paths", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { FontAwesomeSvgIcon } = await import("../FontAwesomeSvgIcon");

        const complexMultiPathIcon = {
            icon: [
                16,
                16,
                [],
                "f004",
                [
                    "M0 0h4v4H0z", // Secondary (40% opacity)
                    "M4 0h4v4H4z", // Primary (100% opacity)
                    "M8 0h4v4H8z"  // Primary (100% opacity)
                ]
            ]
        };

        const { container } = render(
            React.createElement(FontAwesomeSvgIcon, { icon: complexMultiPathIcon })
        );

        const svgElement = container.firstChild as SVGSVGElement;
        const pathElements = svgElement?.getElementsByTagName('path');
        expect(pathElements?.length).toBe(3);

        // First path should have 40% opacity
        expect(pathElements?.[0]?.style.opacity).toBe('0.4');

        // All other paths should have 100% opacity
        expect(pathElements?.[1]?.style.opacity).toBe('1');
        expect(pathElements?.[2]?.style.opacity).toBe('1');
    });
});