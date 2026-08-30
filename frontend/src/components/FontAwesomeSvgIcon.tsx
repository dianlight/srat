import type { SvgIconProps } from "@mui/material/SvgIcon";
import SvgIcon from "@mui/material/SvgIcon";
import * as React from "react";

type FontAwesomeSvgIconProps = SvgIconProps & {
  icon: { icon: [number, number, unknown, unknown, string | string[]] };
};

export const FontAwesomeSvgIcon = React.forwardRef<
  SVGSVGElement,
  FontAwesomeSvgIconProps
>((props, ref) => {
  const {
    icon,
    "data-testid": dataTestId,
    ...rest
  } = props as FontAwesomeSvgIconProps & { "data-testid"?: string };

  const {
    icon: [width, height, , , svgPathData],
  } = icon;

  // Only expose internal path test ids in non-production builds to avoid
  // leaking test metadata and duplicate IDs when multiple icons render.
  const pathTestId =
    process.env.NODE_ENV !== "production" ? "fontawesome-icon-path" : undefined;

  return (
    <SvgIcon
      ref={ref}
      {...rest}
      viewBox={`0 0 ${width} ${height}`}
      data-testid={dataTestId}
    >
      {typeof svgPathData === "string" ? (
        <path data-testid={pathTestId} d={svgPathData} />
      ) : (
        /**
         * A multi-path Font Awesome icon seems to imply a duotune icon. The 0th path seems to
         * be the faded element (referred to as the "secondary" path in the Font Awesome docs)
         * of a duotone icon. 40% is the default opacity.
         *
         * @see https://fontawesome.com/how-to-use/on-the-web/styling/duotone-icons#changing-opacity
         */
        svgPathData.map((d: string, i: number) => (
          <path
            key={d}
            data-testid={pathTestId}
            style={{ opacity: i === 0 ? 0.4 : 1 }}
            d={d}
          />
        ))
      )}
    </SvgIcon>
  );
});
