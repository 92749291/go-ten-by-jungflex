import { NavLink } from "../types";
import { environment } from "../lib/constants";

export const NavLinks: NavLink[] = [
  // {
  //   href: "/",
  //   label: "Bridge",
  //   isExternal: false,
  //   isDropdown: false,
  //   icon: RouteIcon,
  // },
  // {
  //   href: "/transactions",
  //   label: "Transactions",
  //   isExternal: false,
  //   isDropdown: false,
  //   icon: ReaderIcon,
  // },
];

export const apiRoutes = {
  getHealthStatus: `/info/health/`,
  getObscuroConfig: `http://erpc.${environment}.obscu.ro:80/`,
};

export const requestMethods = {
  connectAccounts: "eth_requestAccounts",
  switchNetwork: "wallet_switchEthereumChain",
  addNetwork: "wallet_addEthereumChain",
  getStorageAt: "eth_getStorageAt",
  signTypedData: "eth_signTypedData_v4",
  getChainId: "eth_chainId",
};
