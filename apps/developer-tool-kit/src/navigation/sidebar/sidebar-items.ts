import {
  Activity,
  Code2,
  Container,
  FileJson,
  type LucideIcon,
  Network,
  ScrollText,
  Server,
} from "lucide-react";

export interface NavSubItem {
  title: string;
  url: string;
  icon?: LucideIcon;
  comingSoon?: boolean;
  newTab?: boolean;
  isNew?: boolean;
}

export interface NavMainItem {
  title: string;
  url: string;
  icon?: LucideIcon;
  subItems?: NavSubItem[];
  comingSoon?: boolean;
  newTab?: boolean;
  isNew?: boolean;
}

export interface NavGroup {
  id: number;
  label?: string;
  items: NavMainItem[];
}

export const sidebarItems: NavGroup[] = [
  {
    id: 1,
    label: "Titan Network",
    items: [
      {
        title: "Overview",
        url: "/dashboard/default",
        icon: Network,
      },
      {
        title: "Nodes",
        url: "/dashboard/nodes",
        icon: Server,
      },
      {
        title: "Origin",
        url: "/dashboard/origin",
        icon: FileJson,
      },
      {
        title: "Explorer",
        url: "/dashboard/activity",
        icon: Activity,
      },
    ],
  },
  {
    id: 2,
    label: "Developer",
    items: [
      {
        title: "Contract Studio",
        url: "/dashboard/contracts",
        icon: Code2,
        isNew: true,
      },
      {
        title: "Docker Logs",
        url: "/dashboard/logs",
        icon: ScrollText,
      },
      {
        title: "Containers",
        url: "/dashboard/containers",
        icon: Container,
      },
    ],
  },
];
