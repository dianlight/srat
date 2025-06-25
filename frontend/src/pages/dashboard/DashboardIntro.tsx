import { Card, CardContent, CardHeader, Typography, Box, IconButton, Collapse } from "@mui/material";
import KeyboardArrowLeftIcon from '@mui/icons-material/KeyboardArrowLeft';
import KeyboardArrowRightIcon from '@mui/icons-material/KeyboardArrowRight';

interface DashboardIntroProps {
    isCollapsed: boolean;
    onToggleCollapse: () => void;
}

export function DashboardIntro({ isCollapsed, onToggleCollapse }: DashboardIntroProps) {
    return (
        <Card sx={{
            height: '100%', // Auto height when collapsed, 100% when expanded
            display: 'flex',
            flexDirection: 'column',
        }}>
            {isCollapsed ? (
                // Collapsed view
                <Box sx={{
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center', // Center items horizontally within this box
                    justifyContent: 'flex-start', // Align items to the top
                    height: '100%', // This box should also fill the height
                    width: '100%', // This box should also fill the width
                }}>
                    <IconButton
                        aria-label="expand"
                        onClick={onToggleCollapse}
                        sx={{
                            mb: 1, // Margin below the button
                        }}
                    >
                        <KeyboardArrowRightIcon />
                    </IconButton>
                    <Typography
                        variant="h6"
                        sx={{
                            writingMode: 'vertical-lr', // Text flows top-to-bottom, then left-to-right columns
                            textOrientation: 'upright', // Characters are upright
                            whiteSpace: 'nowrap', // Prevent wrapping
                            flexGrow: 1, // Allow it to take available space and push button/content apart
                            display: 'flex', // Use flexbox for internal centering
                            alignItems: 'center', // Center vertically within its flex item
                            justifyContent: 'center', // Center horizontally within its flex item
                            fontSize: '1rem', // Adjust font size to fit narrow column
                        }}
                    >
                        Welcome to SRAT
                    </Typography>
                </Box>
            ) : (
                // Expanded view
                <>
                    <CardHeader
                        title="Welcome to SRAT"
                        titleTypographyProps={{ variant: 'h6' }}
                        action={ // Button on the right side of the header
                            <IconButton
                                aria-label="collapse"
                                onClick={onToggleCollapse}
                            >
                                <KeyboardArrowLeftIcon />
                            </IconButton>
                        }
                    />
                    <Collapse in={!isCollapsed}> {/* Content collapses/expands */}
                        <CardContent sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column' }}>
                            <Typography variant="body1">
                                This is your storage management dashboard. Here you can get a quick overview of your system's storage health and perform common actions.
                            </Typography>
                            <Box sx={{ flexGrow: 1 }} /> {/* Pushes "Latest News" to the bottom */}
                            <Typography variant="body2" sx={{ mt: 2 }}>
                                <strong>Latest News:</strong> Version X.Y.Z has been released with new performance improvements!
                            </Typography>
                        </CardContent>
                    </Collapse>
                </>
            )}
        </Card>
    );
}