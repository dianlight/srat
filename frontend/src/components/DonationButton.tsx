import { faCoffee } from "@fortawesome/free-solid-svg-icons/faCoffee";
import { faHeart } from "@fortawesome/free-solid-svg-icons/faHeart";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faGithub } from "@fortawesome/free-brands-svg-icons/faGithub";
import VolunteerActivismIcon from "@mui/icons-material/VolunteerActivism";
import {
	IconButton,
	ListItemIcon,
	ListItemText,
	Menu,
	MenuItem,
	Tooltip,
} from "@mui/material";
import { useState } from "react";
import { useGetFundingConfigQuery, type FundingPlatform } from "../store/githubApi";

/**
 * Maps platform names to their corresponding icons
 */
function getPlatformIcon(platform: string): React.ReactNode {
	const iconMap: Record<string, React.ReactNode> = {
		github: <FontAwesomeIcon icon={faGithub} />,
		buy_me_a_coffee: <FontAwesomeIcon icon={faCoffee} />,
		patreon: <FontAwesomeIcon icon={faHeart} />,
		ko_fi: <FontAwesomeIcon icon={faCoffee} />,
		open_collective: <FontAwesomeIcon icon={faHeart} />,
	};

	return iconMap[platform] || <FontAwesomeIcon icon={faHeart} />;
}

/**
 * DonationButton component displays a donation icon with a dropdown menu
 * showing available funding platforms fetched from .github/funding.yaml
 */
export function DonationButton() {
	const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
	const open = Boolean(anchorEl);

	// Fetch funding configuration from GitHub with RTK Query caching
	const { data: fundingPlatforms = [], isLoading, isError, error } = useGetFundingConfigQuery();

	const handleClick = (event: React.MouseEvent<HTMLElement>) => {
		setAnchorEl(anchorEl == null ? event.currentTarget:null);
	};

	const handleClose = () => {
		setAnchorEl(null);
	};

	const handleDonationClick = (platform: FundingPlatform) => {
		window.open(platform.url, "_blank", "noopener,noreferrer");
		handleClose();
	};

	// Don't render if no funding platforms are configured or if there's an error
	if (isLoading || isError || fundingPlatforms.length === 0) {
		if (isError) console.error(error)
		console.log("No funding platforms configured or error fetching data.", isLoading, isError, fundingPlatforms);
		return null;
	}

	return (
		<>
			<IconButton
				sx={{ display: { xs: "inline-flex", sm: "inline-flex" } }}
				size="small"
				onClick={handleClick}
				aria-controls={open ? "donation-menu" : undefined}
				aria-haspopup="true"
				aria-expanded={open ? "true" : undefined}
				data-testid="donation-button"
			>
				<Tooltip title="Support this project" arrow>
					<VolunteerActivismIcon sx={{ color: "white" }} />
				</Tooltip>
			</IconButton>
			<Menu
				id="donation-menu"
				anchorEl={anchorEl}
				open={open}
				onClose={handleClose}
				slotProps={{
					list: {
						"aria-labelledby": "donation-button",
					},
				}}
			>
				{fundingPlatforms.map((platform) => (
					<MenuItem
						key={platform.platform}
						onClick={() => handleDonationClick(platform)}
					>
						<ListItemIcon>{getPlatformIcon(platform.platform)}</ListItemIcon>
						<ListItemText>{platform.label}</ListItemText>
					</MenuItem>
				))}
			</Menu>
		</>
	);
}
