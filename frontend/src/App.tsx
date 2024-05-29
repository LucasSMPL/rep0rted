import { useState } from "react";
import { QueryClient, QueryClientProvider } from "react-query";
import { ThemeProvider } from "./components/theme-provider";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./components/ui/table";
import logo from '/Users/lucas/Desktop/rep0rted/frontend/src/assets/t0_wordmark.svg';
import { Button } from "./components/ui/button";

const queryClient = new QueryClient();

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
    visible: true 
  });

  const [tableData, setTableData] = useState([
    { id: "01", ip: "10.0.101.69", mac: "AE:17:D9:C69", port: "12435" },
  ]);

  const handleSkipRow = () => {
    const newId = (tableData.length + 1).toString().padStart(2, '0');
    setTableData([...tableData, { id: newId, ip: "", mac: "", port: "" }]);
  };

  const handleClearList = () => {
    setTableData([]);
  };

  const handleExportList = () => {
    const csvHeader = "ID,IP,MAC,Port\n";
    const csvRows = tableData.map(row => `${row.id},${row.ip},${row.mac},${row.port}`).join("\n");
    const csvContent = csvHeader + csvRows;
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    link.setAttribute('href', url);
    link.setAttribute('download', 'table_data.csv');
    link.style.visibility = 'hidden';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <div className="flex flex-col h-screen overflow-hidden">
      {banner.visible && (
        <div className="text-center py-4 border" style={{ backgroundColor: "#0a0a0a", color: "#ffffff", borderColor: "#ee821a" }}>
          {banner.message}
        </div>
      )}
      <div className="flex items-center justify-center" id="logo-container">
        <img src={logo} alt="Logo" className="max-w-sm p-5"/>
      </div>
      <div className="flex items-center justify-center" id="header-text">
        <p>Currently listening for ASIC IP Addresses. Press the IP Report button on your miner, and check the table below.</p>
        <span className="blinking-circle"></span>
      </div>
      <div className="flex items-center justify-center pt-5" id="header-text">
        <p>(Port 14235 = Antminer) & (Port 8888 = Whatsminer) & (Port 12345 = Aurdaine)</p>
      </div>
      <div className="flex items-center justify-center p-5" id="button-section">
        <Button className="m-2" onClick={handleSkipRow} style={{ background: 'linear-gradient(90deg, hsla(4, 93%, 67%, 1) 0%, hsla(29, 86%, 52%, 1) 100%)', border: 'none', color: 'white' }}>
          Skip Row
        </Button>
        <Button className="m-2" onClick={handleClearList} style={{ background: 'linear-gradient(90deg, hsla(4, 93%, 67%, 1) 0%, hsla(29, 86%, 52%, 1) 100%)', border: 'none', color: 'white' }}>
          Clear List
        </Button>
        <Button className="m-2" onClick={handleExportList} style={{ background: 'linear-gradient(90deg, hsla(4, 93%, 67%, 1) 0%, hsla(29, 86%, 52%, 1) 100%)', border: 'none', color: 'white' }}>
          Export List
        </Button>
      </div>
      <div className="flex items-center justify-center p-10">
        <div className="max-w-2xl w-full">
          <Table className="w-full">
            <TableHeader style={{backgroundColor: "black"}}>
              <TableRow>
                <TableHead className="text-center w-[100px]">ID</TableHead>
                <TableHead className="text-center">IP</TableHead>
                <TableHead className="text-center">MAC</TableHead>
                <TableHead className="text-center">Port</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tableData.map((row, index) => (
                <TableRow key={index}>
                  {/* I want to inverse the ID Indexing. Sort descending. */}
                  <TableCell className="text-center font-medium">{row.id}</TableCell>
                  <TableCell className="text-center">{row.ip}</TableCell>
                  <TableCell className="text-center">{row.mac}</TableCell>
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
