import {
  BarChart3,
  Building2,
  Cable,
  ChevronLeft,
  Database,
  Headphones,
  HeartPulse,
  LayoutDashboard,
  ListTree,
  Network,
  Send,
  Server,
} from 'lucide-react';
import { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { ServerStatus } from '@/app/ServerStatus';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';

interface NavItemProps {
  name: string;
  path: string;
  icon: React.ReactNode;
  small: boolean;
  isActive: boolean;
  exact?: boolean;
}

const NavItem = ({ name, path, icon, small, isActive }: NavItemProps) => {
  return (
    <Link
      to={path}
      className={`p-3 rounded-lg flex items-center gap-3 transition-all duration-200 ease-in-out group overflow-hidden ${
        isActive
          ? 'bg-primary/5 text-primary font-medium'
          : 'hover:bg-muted text-muted-foreground'
      }`}
    >
      <span
        className={`transform transition-transform duration-300 ${
          small ? 'scale-110' : 'scale-100'
        } ${isActive ? 'text-primary' : 'text-muted-foreground'}`}
      >
        {icon}
      </span>
      <span
        className={`whitespace-nowrap transition-all duration-300 ${
          small ? 'opacity-0 translate-x-10' : 'opacity-100 translate-x-0'
        }`}
      >
        {name}
      </span>
    </Link>
  );
};

const Sidebar = () => {
  const location = useLocation();
  // Start collapsed (icons only) on phone-sized screens.
  const [small, setSmall] = useState(
    () => window.matchMedia('(max-width: 767px)').matches
  );

  const navItems = [
    {
      name: 'Overview',
      path: '/dashboard',
      icon: <LayoutDashboard className="h-5 w-5" />,
      exact: true,
    },
    {
      name: 'Health',
      path: '/dashboard/health',
      icon: <HeartPulse className="h-5 w-5" />,
      exact: false,
    },
    {
      name: 'Subjects',
      path: '/dashboard/subjects',
      icon: <ListTree className="h-5 w-5" />,
      exact: false,
    },
    {
      name: 'Topology',
      path: '/dashboard/topology',
      icon: <Network className="h-5 w-5" />,
      exact: false,
    },
    {
      name: 'Servers',
      path: '/dashboard/servers',
      icon: <Server className="h-5 w-5" />,
      exact: false,
    },
    {
      name: 'Connections',
      path: '/dashboard/connections',
      icon: <Cable className="h-5 w-5" />,
      exact: false,
    },
    {
      name: 'Accounts',
      path: '/dashboard/accounts',
      icon: <Building2 className="h-5 w-5" />,
      exact: false,
    },
    {
      name: 'Streams',
      path: '/dashboard/streams',
      icon: <BarChart3 className="h-5 w-5" />,
      exact: false,
    },
    {
      name: 'Key-Value Store',
      path: '/dashboard/kv',
      icon: <Database className="h-5 w-5" />,
      exact: false,
    },
    {
      name: 'Publish',
      path: '/dashboard/publish',
      icon: <Send className="h-5 w-5" />,
      exact: true,
    },
    {
      name: 'Subscribe',
      path: '/dashboard/subscribe',
      icon: <Headphones className="h-5 w-5" />,
      exact: true,
    },
  ];

  const bottomNavItems: NavItemProps[] = [];

  const isActive = (itemPath: string, exact = false) => {
    return exact
      ? location.pathname === itemPath
      : location.pathname.startsWith(itemPath);
  };

  return (
    <div
      className={`h-screen bg-card flex-none relative transition-all duration-300 ease-in-out border-r border-border ${
        small ? 'w-[70px]' : 'w-[250px]'
      }`}
    >
      <button
        type="button"
        onClick={() => setSmall(!small)}
        className={`absolute -right-3 z-10 top-[90px] bg-card rounded-full p-1.5 border shadow-md cursor-pointer hover:bg-muted transition-all duration-300 ease-in-out ${
          small ? 'rotate-180' : ''
        }`}
      >
        <ChevronLeft className="h-4 w-4 text-muted-foreground" />
      </button>

      <div className="flex flex-col gap-5 h-full">
        <div className="h-[70px] flex items-center justify-center mt-5 overflow-hidden">
          <div
            className={`transition-all duration-300 ease-in-out transform ${
              small ? 'scale-90' : 'scale-100'
            }`}
          >
            {small ? (
              <div className="w-8 h-8 bg-primary rounded-full flex items-center justify-center">
                <img
                  src="/heynats.jpg"
                  className="w-full h-full object-cover"
                  alt="HeyNATS logo"
                />
              </div>
            ) : (
              <div className="flex items-center">
                <div className="w-8 h-8 bg-primary rounded-full flex items-center justify-center overflow-hidden mr-3">
                  <img
                    src="/heynats.jpg"
                    className="w-full h-full object-cover"
                    alt="HeyNATS logo"
                  />
                </div>
                <h1 className="text-xl font-semibold text-foreground">
                  Hey NATS
                </h1>
              </div>
            )}
          </div>
        </div>

        <div className="flex-1 overflow-y-auto flex flex-col gap-2 px-3 mt-10">
          {navItems.map((item) => (
            <TooltipProvider key={item.path} delayDuration={100}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <div>
                    <NavItem
                      name={item.name}
                      path={item.path}
                      icon={item.icon}
                      small={small}
                      isActive={isActive(item.path, item.exact)}
                    />
                  </div>
                </TooltipTrigger>
                {small && (
                  <TooltipContent
                    side="right"
                    align="center"
                    className="z-50 text-base"
                  >
                    <p>{item.name}</p>
                  </TooltipContent>
                )}
              </Tooltip>
            </TooltipProvider>
          ))}
        </div>

        <div className="flex-1 max-h-[150px] flex flex-col gap-2 px-3 justify-end mb-5">
          {bottomNavItems.map((item) => (
            <TooltipProvider key={item.path} delayDuration={100}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <div>
                    <NavItem
                      name={item.name}
                      path={item.path}
                      icon={item.icon}
                      small={small}
                      isActive={isActive(item.path)}
                    />
                  </div>
                </TooltipTrigger>
                {small && (
                  <TooltipContent
                    side="right"
                    align="center"
                    className="z-50 text-base"
                  >
                    <p>{item.name}</p>
                  </TooltipContent>
                )}
              </Tooltip>
            </TooltipProvider>
          ))}
          <ServerStatus compact={small} />
        </div>
      </div>
    </div>
  );
};

export { Sidebar };
