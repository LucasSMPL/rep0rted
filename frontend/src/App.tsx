import { useEffect, useState } from "react";
import { QueryClient, QueryClientProvider } from "react-query";
import logo from "./assets/t0_wordmark.svg";
import { ThemeProvider } from "./components/theme-provider";
import { Button } from "./components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "./components/ui/table";

const queryClient = new QueryClient();

interface PacketData {
  id: number;
  ip_src: string;
  mac_src: string;
  port: number;
  type: string;
  rate_ideal: number;
}

function App() {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      <QueryClientProvider client={queryClient}>
        <Reporter />
      </QueryClientProvider>
    </ThemeProvider>
  );
}

export default App;

function Reporter() {
  const [banner] = useState({
    message: "The rep0rter app is up to date!",
    visible: true,
  });

  const [tableData, setTableData] = useState<PacketData[]>([]);
  const [nextId, setNextId] = useState(1);

  useEffect(() => {
    const eventSource = new EventSource("http://localhost:7070/events");

    eventSource.onmessage = function (event) {
      const newPacket: PacketData = JSON.parse(event.data);
      newPacket.id = nextId;
      setTableData((prevData) => [newPacket, ...prevData]);
      setNextId(nextId + 1);
    };

    return () => {
      eventSource.close();
    };
  }, [nextId]);

  const handleSkipRow = () => {
    setTableData([
      { id: nextId, ip_src: "", mac_src: "", type: "", rate_ideal: 0, port: 0 },
      ...tableData,
    ]);
    setNextId(nextId + 1);
  };

  const handleClearList = async () => {
    try {
      const response = await fetch("http://localhost:7070/clear", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
      });
      if (response.ok) {
        setTableData([]);
        setNextId(1);
        console.log("Successfully cleared the backend cache");
      } else {
        console.error("Failed to clear the backend cache");
      }
    } catch (error) {
      console.error("Error clearing the backend cache:", error);
    }
  };

  const handleExportList = () => {
    const csvHeader = "ID,IP,MAC,Type,Port\n";
    const csvRows = tableData
      .map(
        (row) =>
          `${row.id},${row.ip_src},${row.mac_src},${row.type} - ${row.rate_ideal} TH,${row.port}`
      )
      .join("\n");
    const csvContent = csvHeader + csvRows;
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
    const link = document.createElement("a");
    const url = URL.createObjectURL(blob);
    link.setAttribute("href", url);
    link.setAttribute("download", "rep0rter_data.csv");
    link.style.visibility = "hidden";
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <div className="flex flex-col h-screen overflow-hidden">
      {banner.visible && (
        <div
          className="text-center py-4 border"
          style={{
            backgroundColor: "#0a0a0a",
            color: "#ffffff",
            borderColor: "#ee821a",
          }}
        >
          {banner.message}
        </div>
      )}
      <div className="flex items-center justify-center" id="logo-container">
        <img src={logo} alt="Logo" className="max-w-sm p-5" />
      </div>
      <div className="flex items-center justify-center" id="header-text">
        <p>
          Currently listening for ASIC IP Addresses. Press the IP Report button
          on your miner, and check the table below.
        </p>
        <span className="blinking-circle"></span>
      </div>
      <div className="flex items-center justify-center pt-5" id="header-text">
        <p>
          (Port 14235 = Antminer) & (Port 8888 = Whatsminer) & (Port 12345 =
          Aurdaine) & (Port 60040 = IceRiver)
        </p>
      </div>
      <div className="flex items-center justify-center p-5" id="button-section">
        <Button
          className="m-2"
          onClick={handleSkipRow}
          style={{
            background:
              "linear-gradient(90deg, hsla(4, 93%, 67%, 1) 0%, hsla(29, 86%, 52%, 1) 100%)",
            border: "none",
            color: "white",
          }}
        >
          Skip Row
        </Button>
        <Button
          className="m-2"
          onClick={handleClearList}
          style={{
            background:
              "linear-gradient(90deg, hsla(4, 93%, 67%, 1) 0%, hsla(29, 86%, 52%, 1) 100%)",
            border: "none",
            color: "white",
          }}
        >
          Clear List
        </Button>
        <Button
          className="m-2"
          onClick={handleExportList}
          style={{
            background:
              "linear-gradient(90deg, hsla(4, 93%, 67%, 1) 0%, hsla(29, 86%, 52%, 1) 100%)",
            border: "none",
            color: "white",
          }}
        >
          Export List
        </Button>
      </div>
      <div className="flex items-center justify-center p-10 w-full">
        <div className="max-w-2xl w-full overflow-y-auto max-h-96">
          <Table className="w-full">
            <TableHeader className="sticky top-0 bg-black z-10">
              <TableRow>
                <TableHead className="text-center w-[100px]">ID</TableHead>
                <TableHead className="text-center">IP</TableHead>
                <TableHead className="text-center">MAC</TableHead>
                <TableHead className="text-center">Type</TableHead>
                <TableHead className="text-center">Port</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tableData.map((row, index) => (
                <TableRow key={index}>
                  <TableCell className="text-center font-medium">
                    {row.id}
                  </TableCell>
                  <TableCell className="text-center">
                    <a
                      href={`http://root:root@${row.ip_src}`}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {row.ip_src}
                    </a>
                  </TableCell>
                  <TableCell className="text-center">{row.mac_src}</TableCell>
                  <TableCell className="text-center">
                    {row.type} ({row.rate_ideal} TH)
                  </TableCell>
                  <TableCell className="text-center">{row.port}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  );
}
