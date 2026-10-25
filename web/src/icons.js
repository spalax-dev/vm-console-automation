/**
 * Registro de los iconos Lucide que usa la interfaz.
 *
 * Se importan por nombre para que el empaquetador incluya solo los glifos
 * necesarios: importar el paquete completo añade más de 400 kB al binario.
 */

import {
  Activity,
  AlertTriangle,
  Check,
  CheckCircle2,
  Copy,
  Download,
  Eye,
  FileDown,
  FileText,
  GitMerge,
  History,
  Home,
  Info,
  KeyRound,
  LayoutGrid,
  List,
  Monitor,
  Package,
  Play,
  Plus,
  PlusCircle,
  Power,
  PowerOff,
  RefreshCw,
  RotateCcw,
  Save,
  Search,
  Server,
  Settings,
  SlidersHorizontal,
  Square,
  Trash2,
  Upload,
  UserMinus,
  UserPlus,
  Users,
  X,
  XCircle,
  createIcons,
} from 'lucide';

/** Iconos disponibles, indexados por su nombre en PascalCase. */
export const icons = {
  Activity,
  AlertTriangle,
  Check,
  CheckCircle2,
  Copy,
  Download,
  Eye,
  FileDown,
  FileText,
  GitMerge,
  History,
  Home,
  Info,
  KeyRound,
  LayoutGrid,
  List,
  Monitor,
  Package,
  Play,
  Plus,
  PlusCircle,
  Power,
  PowerOff,
  RefreshCw,
  RotateCcw,
  Save,
  Search,
  Server,
  Settings,
  SlidersHorizontal,
  Square,
  Trash2,
  Upload,
  UserMinus,
  UserPlus,
  Users,
  X,
  XCircle,
};

/**
 * Reemplaza los elementos `[data-lucide]` de un contenedor por su SVG.
 *
 * @param {HTMLElement|Document} [raiz] Contenedor donde se buscan los iconos.
 */
export function renderIcons(raiz = document) {
  if (!raiz) {
    return;
  }

  createIcons({
    icons,
    attrs: {
      'stroke-width': 1.5,
      width: 16,
      height: 16,
    },
    nameAttr: 'data-lucide',
    root: raiz,
  });
}
